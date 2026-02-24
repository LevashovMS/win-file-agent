package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

const (
	// 30 seconds was chosen as it's the
	// same duration as http.DefaultTransport's timeout.
	DefaultDialTimeout = 30 * time.Second
)

// EntryType describes the different types of an Entry.
type EntryType int

// The differents types of an Entry
const (
	EntryTypeFile EntryType = iota
	EntryTypeFolder
	EntryTypeLink
)

// TransferType denotes the formats for transferring Entries.
type TransferType string

// The different transfer types
const (
	TransferTypeBinary = TransferType("I")
	TransferTypeASCII  = TransferType("A")
)

// Time format used by the MDTM and MFMT commands
const timeFormat = "20060102150405"

// ServerConn represents the connection to a remote FTP server.
// A single connection only supports one in-flight data connection.
// It is not safe to be called concurrently.
type ServerConn struct {
	options *dialOptions
	conn    *textproto.Conn // connection wrapper for text protocol
	netConn net.Conn        // underlying network connection
	host    string

	// Server capabilities discovered at runtime
	features      map[string]string
	skipEPSV      bool
	mlstSupported bool
	mfmtSupported bool
	mdtmSupported bool
	mdtmCanWrite  bool
	usePRET       bool
}

// DialOption represents an option to start a new connection with Dial
type DialOption struct {
	setup func(do *dialOptions)
}

// dialOptions contains all the options set by DialOption.setup
type dialOptions struct {
	context         context.Context
	dialer          net.Dialer
	tlsConfig       *tls.Config
	explicitTLS     bool
	disableEPSV     bool
	disableUTF8     bool
	disableMLSD     bool
	writingMDTM     bool
	forceListHidden bool
	location        *time.Location
	debugOutput     io.Writer
	dialFunc        func(network, address string) (net.Conn, error)
	shutTimeout     time.Duration // time to wait for data connection closing status
}

// Entry describes a file and is returned by List().
type Entry struct {
	Name   string
	Target string // target of symbolic link
	Type   EntryType
	Size   uint64
	Time   time.Time
}

// Response represents a data-connection
type Response struct {
	conn   net.Conn
	c      *ServerConn
	closed bool
}

// Dial connects to the specified address with optional options
func Dial(addr string, options ...DialOption) (*ServerConn, error) {
	do := &dialOptions{}
	for _, option := range options {
		option.setup(do)
	}

	if do.location == nil {
		do.location = time.UTC
	}

	dialFunc := do.dialFunc

	if dialFunc == nil {
		ctx := do.context

		if ctx == nil {
			ctx = context.Background()
		}
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, DefaultDialTimeout)
			defer cancel()
		}

		if do.tlsConfig != nil && !do.explicitTLS {
			dialFunc = func(network, address string) (net.Conn, error) {
				tlsDialer := &tls.Dialer{
					NetDialer: &do.dialer,
					Config:    do.tlsConfig,
				}
				return tlsDialer.DialContext(ctx, network, addr)
			}
		} else {

			dialFunc = func(network, address string) (net.Conn, error) {
				return do.dialer.DialContext(ctx, network, addr)
			}
		}
	}

	tconn, err := dialFunc("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Use the resolved IP address in case addr contains a domain name
	// If we use the domain name, we might not resolve to the same IP.
	remoteAddr := tconn.RemoteAddr().(*net.TCPAddr)

	c := &ServerConn{
		options:  do,
		features: make(map[string]string),
		conn:     textproto.NewConn(do.wrapConn(tconn)),
		netConn:  tconn,
		host:     remoteAddr.IP.String(),
	}

	_, _, err = c.conn.ReadResponse(StatusReady)
	if err != nil {
		_ = c.Quit()
		return nil, err
	}

	if do.explicitTLS {
		if err := c.authTLS(); err != nil {
			_ = c.Quit()
			return nil, err
		}
		tconn = tls.Client(tconn, do.tlsConfig)
		c.conn = textproto.NewConn(do.wrapConn(tconn))
	}

	return c, nil
}

// DialWithContext returns a DialOption that configures the ServerConn with specified context
// The context will be used for the initial connection setup
func DialWithContext(ctx context.Context) DialOption {
	return DialOption{func(do *dialOptions) {
		do.context = ctx
	}}
}

// DialWithDialFunc returns a DialOption that configures the ServerConn to use the
// specified function to establish both control and data connections
//
// If used together with the DialWithNetConn option, the DialWithNetConn
// takes precedence for the control connection, while data connections will
// be established using function specified with the DialWithDialFunc option
func DialWithDialFunc(f func(network, address string) (net.Conn, error)) DialOption {
	return DialOption{func(do *dialOptions) {
		do.dialFunc = f
	}}
}

func (o *dialOptions) wrapConn(netConn net.Conn) io.ReadWriteCloser {
	if o.debugOutput == nil {
		return netConn
	}

	return newDebugWrapper(netConn, o.debugOutput)
}

// Login authenticates the client with specified user and password.
//
// "anonymous"/"anonymous" is a common user/password scheme for FTP servers
// that allows anonymous read-only accounts.
func (c *ServerConn) Login(user, password string) error {
	code, message, err := c.cmd(-1, "USER %s", user)
	if err != nil {
		return err
	}

	switch code {
	case StatusLoggedIn:
	case StatusUserOK:
		_, _, err = c.cmd(StatusLoggedIn, "PASS %s", password)
		if err != nil {
			return err
		}
	default:
		return errors.New(message)
	}

	// Probe features
	err = c.feat()
	if err != nil {
		return err
	}
	if _, mlstSupported := c.features["MLST"]; mlstSupported && !c.options.disableMLSD {
		c.mlstSupported = true
	}
	_, c.usePRET = c.features["PRET"]

	_, c.mfmtSupported = c.features["MFMT"]
	_, c.mdtmSupported = c.features["MDTM"]
	c.mdtmCanWrite = c.mdtmSupported && c.options.writingMDTM

	// Switch to binary mode
	if err = c.Type(TransferTypeBinary); err != nil {
		return err
	}

	// Switch to UTF-8
	if !c.options.disableUTF8 {
		err = c.setUTF8()
	}

	// If using implicit TLS, make data connections also use TLS
	if c.options.tlsConfig != nil {
		if _, _, err = c.cmd(StatusCommandOK, "PBSZ 0"); err != nil {
			return err
		}
		if _, _, err = c.cmd(StatusCommandOK, "PROT P"); err != nil {
			return err
		}
	}

	return err
}

// authTLS upgrades the connection to use TLS
func (c *ServerConn) authTLS() error {
	_, _, err := c.cmd(StatusAuthOK, "AUTH TLS")
	return err
}

// feat issues a FEAT FTP command to list the additional commands supported by
// the remote FTP server.
// FEAT is described in RFC 2389
func (c *ServerConn) feat() error {
	code, message, err := c.cmd(-1, "FEAT")
	if err != nil {
		return err
	}

	if code != StatusSystem {
		// The server does not support the FEAT command. This is not an
		// error: we consider that there is no additional feature.
		return nil
	}

	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, " ") {
			continue
		}

		line = strings.TrimSpace(line)
		featureElements := strings.SplitN(line, " ", 2)

		command := featureElements[0]

		var commandDesc string
		if len(featureElements) == 2 {
			commandDesc = featureElements[1]
		}

		c.features[command] = commandDesc
	}

	return nil
}

// setUTF8 issues an "OPTS UTF8 ON" command.
func (c *ServerConn) setUTF8() error {
	if _, ok := c.features["UTF8"]; !ok {
		return nil
	}

	code, message, err := c.cmd(-1, "OPTS UTF8 ON")
	if err != nil {
		return err
	}

	// Workaround for FTP servers, that does not support this option.
	if code == StatusBadArguments || code == StatusNotImplementedParameter {
		return nil
	}

	// The ftpd "filezilla-server" has FEAT support for UTF8, but always returns
	// "202 UTF8 mode is always enabled. No need to send this command." when
	// trying to use it. That's OK
	if code == StatusCommandNotImplemented {
		return nil
	}

	if code != StatusCommandOK {
		return errors.New(message)
	}

	return nil
}

// epsv issues an "EPSV" command to get a port number for a data connection.
func (c *ServerConn) epsv() (port int, err error) {
	_, line, err := c.cmd(StatusExtendedPassiveMode, "EPSV")
	if err != nil {
		return 0, err
	}

	start := strings.Index(line, "|||")
	end := strings.LastIndex(line, "|")
	if start == -1 || end == -1 {
		return 0, errors.New("invalid EPSV response format")
	}
	port, err = strconv.Atoi(line[start+3 : end])
	return port, err
}

// pasv issues a "PASV" command to get a port number for a data connection.
func (c *ServerConn) pasv() (host string, port int, err error) {
	_, line, err := c.cmd(StatusPassiveMode, "PASV")
	if err != nil {
		return "", 0, err
	}

	// PASV response format : 227 Entering Passive Mode (h1,h2,h3,h4,p1,p2).
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start == -1 || end == -1 {
		return "", 0, errors.New("invalid PASV response format")
	}

	// We have to split the response string
	pasvData := strings.Split(line[start+1:end], ",")

	if len(pasvData) < 6 {
		return "", 0, errors.New("invalid PASV response format")
	}

	// Let's compute the port number
	portPart1, err := strconv.Atoi(pasvData[4])
	if err != nil {
		return "", 0, err
	}

	portPart2, err := strconv.Atoi(pasvData[5])
	if err != nil {
		return "", 0, err
	}

	// Recompose port
	port = portPart1*256 + portPart2

	// Make the IP address to connect to
	host = strings.Join(pasvData[0:4], ".")

	if c.host != host {
		if cmdIP := net.ParseIP(c.host); cmdIP != nil {
			if dataIP := net.ParseIP(host); dataIP != nil {
				if isBogusDataIP(cmdIP, dataIP) {
					return c.host, port, nil
				}
			}
		}
	}
	return host, port, nil
}

func isBogusDataIP(cmdIP, dataIP net.IP) bool {
	// Logic stolen from lftp (https://github.com/lavv17/lftp/blob/d67fc14d085849a6b0418bb3e912fea2e94c18d1/src/ftpclass.cc#L769)
	return dataIP.IsMulticast() ||
		cmdIP.IsPrivate() != dataIP.IsPrivate() ||
		cmdIP.IsLoopback() != dataIP.IsLoopback()
}

// getDataConnPort returns a host, port for a new data connection
// it uses the best available method to do so
func (c *ServerConn) getDataConnPort() (string, int, error) {
	if !c.options.disableEPSV && !c.skipEPSV {
		if port, err := c.epsv(); err == nil {
			return c.host, port, nil
		}

		// if there is an error, skip EPSV for the next attempts
		c.skipEPSV = true
	}

	return c.pasv()
}

// openDataConn creates a new FTP data connection.
func (c *ServerConn) openDataConn() (net.Conn, error) {
	host, port, err := c.getDataConnPort()
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	if c.options.dialFunc != nil {
		return c.options.dialFunc("tcp", addr)
	}

	if c.options.tlsConfig != nil {
		// We don't use tls.DialWithDialer here (which does Dial, create
		// the Client and then do the Handshake) because it seems to
		// hang with some FTP servers, namely proftpd and pureftpd.
		//
		// Instead we do Dial, create the Client and wait for the first
		// Read or Write to trigger the Handshake.
		//
		// This means that if we are uploading a zero sized file, we
		// need to make sure we do the Handshake explicitly as Write
		// won't have been called. This is done in StorFrom().
		//
		// See: https://github.com/jlaffaye/ftp/issues/282
		conn, err := c.options.dialer.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		tlsConn := tls.Client(conn, c.options.tlsConfig)
		return tlsConn, nil
	}

	return c.options.dialer.Dial("tcp", addr)
}

// cmd is a helper function to execute a command and check for the expected FTP
// return code
func (c *ServerConn) cmd(expected int, format string, args ...any) (int, string, error) {
	_, err := c.conn.Cmd(format, args...)
	if err != nil {
		return 0, "", err
	}

	return c.conn.ReadResponse(expected)
}

// cmdDataConnFrom executes a command which require a FTP data connection.
// Issues a REST FTP command to specify the number of bytes to skip for the transfer.
func (c *ServerConn) cmdDataConnFrom(offset uint64, format string, args ...any) (net.Conn, error) {
	// If server requires PRET send the PRET command to warm it up
	// See: https://tools.ietf.org/html/draft-dd-pret-00
	if c.usePRET {
		_, _, err := c.cmd(-1, "PRET "+format, args...)
		if err != nil {
			return nil, err
		}
	}

	conn, err := c.openDataConn()
	if err != nil {
		return nil, err
	}

	if offset != 0 {
		_, _, err = c.cmd(StatusRequestFilePending, "REST %d", offset)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	_, err = c.conn.Cmd(format, args...)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	code, msg, err := c.conn.ReadResponse(-1)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if code != StatusAlreadyOpen && code != StatusAboutToSend {
		_ = conn.Close()
		return nil, &textproto.Error{Code: code, Msg: msg}
	}

	return conn, nil
}

// Type switches the transfer mode for the connection.
func (c *ServerConn) Type(transferType TransferType) (err error) {
	_, _, err = c.cmd(StatusCommandOK, "TYPE "+string(transferType))
	return err
}

// Stor issues a STOR FTP command to store a file to the remote FTP server.
// Stor creates the specified file with the content of the io.Reader.
//
// Hint: io.Pipe() can be used if an io.Writer is required.
func (c *ServerConn) Stor(path string, r io.Reader) error {
	return c.StorFrom(path, r, 0)
}

// checkDataShut reads the "closing data connection" status from the
// control connection. It is called after transferring a piece of data
// on the data connection during which the control connection was idle.
// This may result in the idle timeout triggering on the control connection
// right when we try to read the response.
// The ShutTimeout dial option will rescue here. It will nudge the control
// connection deadline right before checking the data closing status.
func (c *ServerConn) checkDataShut() error {
	if c.options.shutTimeout != 0 {
		shutDeadline := time.Now().Add(c.options.shutTimeout)
		if err := c.netConn.SetDeadline(shutDeadline); err != nil {
			return err
		}
	}
	_, _, err := c.conn.ReadResponse(StatusClosingDataConnection)
	return err
}

// StorFrom issues a STOR FTP command to store a file to the remote FTP server.
// Stor creates the specified file with the content of the io.Reader, writing
// on the server will start at the given file offset.
//
// Hint: io.Pipe() can be used if an io.Writer is required.
func (c *ServerConn) StorFrom(path string, r io.Reader, offset uint64) error {
	conn, err := c.cmdDataConnFrom(offset, "STOR %s", path)
	if err != nil {
		return err
	}

	var errs error

	// if the upload fails we still need to try to read the server
	// response otherwise if the failure is not due to a connection problem,
	// for example the server denied the upload for quota limits, we miss
	// the response and we cannot use the connection to send other commands.
	if n, err := io.Copy(conn, r); err != nil {
		errs = errors.Join(errs, err)
	} else if n == 0 {
		// If we wrote no bytes and got no error, make sure we call
		// tls.Handshake on the connection as it won't get called
		// unless Write() is called. (See comment in openDataConn()).
		//
		// ProFTP doesn't like this and returns "Unable to build data
		// connection: Operation not permitted" when trying to upload
		// an empty file without this.
		if do, ok := conn.(interface{ Handshake() error }); ok {
			if err := do.Handshake(); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}

	if err := conn.Close(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := c.checkDataShut(); err != nil {
		errs = errors.Join(errs, err)
	}

	return errs
}

// Quit issues a QUIT FTP command to properly close the connection from the
// remote FTP server.
func (c *ServerConn) Quit() error {
	var errs error

	if _, err := c.conn.Cmd("QUIT"); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := c.conn.Close(); err != nil {
		errs = errors.Join(errs, err)
	}

	return errs
}

// Close implements the io.Closer interface on a FTP data connection.
// After the first call, Close will do nothing and return nil.
func (r *Response) Close() error {
	if r.closed {
		return nil
	}

	var errs error

	if err := r.conn.Close(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := r.c.checkDataShut(); err != nil {
		errs = errors.Join(errs, err)
	}

	r.closed = true
	return errs
}

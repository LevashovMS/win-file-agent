#!/bin/bash

# params
URL="http://91.220.62.199:8080/v1"
REQ_DATA='{\"in_dir\":\"C:\\\\Users\\\\Administrator\\\\Downloads\\\\InDir\",\"urls\":[%s],\"cmd\":\"ffmpeg.exe\",\"args\":[\"-i\",\"{input}\",\"-c:v\",\"libx264\",\"-b:v\",\"500k\",\"-c:a\",\"copy\",\"{output}\"],\"out_ext\":\"mp4\",\"ftp\":{\"addr\":\"storage007.noisypeak.com:21\",\"login\":\"\",\"pass\":\"\"}}'
CSV="CS100files.txt"
TASK_COUNT=1
FILE_COUNT=2

# controllers
TASK="task"

echo "Начало"

run_task() {
    for i in {1..$1}; do
        echo "Номер: $i"
    done
}

task() {
    echo "Выполнение задачи $1 $REQ_DATA"
    sleep 2
}

function req_post() {
    local arr=("$@")
    formatted_string=''
    first=true
    for each in "${arr[@]}"
    do
        if $first; then
            first=false
            formatted_string+='"'$each'"'
        else
            formatted_string+=',"'$each'"'
        fi
    done

    #echo "${formatted_string}"

    printf -v post_data $REQ_DATA $formatted_string
    echo $post_data

    local url=$URL'/'$TASK
    echo $url
    sleep 3

    local tmpfile=$(mktemp)

    # Run curl, writing the body to a temporary file and the status code to stdout
    local status_code=$(curl -s -w "%{http_code}" -o "$tmpfile" -d "$post_data" "$url")
    local body=$(cat "$tmpfile")
    rm "$tmpfile" # Clean up the temporary file

    echo "Status Code: $status_code"
    echo "Response Body: $body"

    if [ $status_code -eq 201 ]; then
        req_get $body
    fi
}

function req_get() {
    while true; do
        url=$URL'/'$TASK'/'$1
        curl -i --location $url
        # Run curl, writing the body to a temporary file and the status code to stdout
        local status_code=$(curl -s -w "%{http_code}" -o "$tmpfile" "$url")

        local body=$(cat "$tmpfile")
        rm "$tmpfile" # Clean up the temporary file
        echo "Status Code: $status_code"

        if [ $status_code -eq 200 ]; then
            echo "Response Body: $body"
        fi

        sleep 3

        return
    done
}

function read_file() {
    # Check if the file exists (optional, but good practice)
    if [ -f "$CSV" ]; then
        # Read the file line by line
        idx=0
        tasks=0
        array=()
        while IFS= read -r line; do
            idx=$((idx+1))
            # echo "Line content: $line"
            array+=($line)
            if [ $((idx % FILE_COUNT)) -eq 0 ]; then
                #echo "$array"
                req_post "${array[@]}" &
                tasks=$((tasks+1))
                array=()
            fi
            if [ $tasks -eq $TASK_COUNT ]; then
                sleep 1
                tasks=0
                echo "WAIT"
                wait
            fi
        done < "$CSV"
    else
        echo "Error: File '$CSV' not found."
    fi
}

function main() {
    echo "Нажмите Ctrl+C для выхода"
    while true; do
        read_file
        echo "Полный проход по файлам"
        sleep 5
    done
}

main

echo "Все задачи завершены"

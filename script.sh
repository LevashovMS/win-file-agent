#!/bin/bash

# params
# ссылка на сервис + версия
URL="http://91.220.62.199:8080/v1"
# REQ_DATA это формат данных с тремя неизвестными:
# \"urls\":[%s] - собираются из файла переменной CSV
# \"login\":\"%s\",\"pass\":\"%s\" - логин и пароль для подключения к ftp, передаются как два первых параметра в скрипте при запуске
REQ_DATA='{\"in_dir\":\"C:\\\\Users\\\\Administrator\\\\Downloads\\\\InDir\",\"urls\":[%s],\"cmd\":\"ffmpeg.exe\",\"args\":[\"-i\",\"{input}\",\"-c:v\",\"copy\",\"-c:a\",\"aac\",\"-b:a\",\"64k\",\"{output}\"],\"out_ext\":\"mp4\",\"ftp\":{\"addr\":\"storage007.noisypeak.com:21\",\"login\":\"%s\",\"pass\":\"%s\"}}'
# файл в котором лежат url откуда качать видео
CSV="CS100files.txt"
# кол-во одновременных выполнение задач
TASK_COUNT=1
# кол-во файлов в задаче
FILE_COUNT=2

# controllers
TASK="task"

#arguments
LOGIN=$1
PASS=$2

function get_state_string() {
    case "$1" in
        0   ) echo "CREATE";;
        1   ) echo "DOWNLOAD";;
        2   ) echo "PROCESS";;
        3   ) echo "SAVING";;
        4   ) echo "CANCEL";;
        5   ) echo "FINISH";;
        127   ) echo "ERROR";;
    esac
}

# Принимает:
# длинну массива
# массив
# состояние отслеживания
function req_post() {
    local array_len="$1"
    shift

    local arr=("${@:1:$array_len}")
    shift "$((array_len))"

    local stop_state="$1"

    local formatted_string=''
    local first=true

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

    printf -v post_data $REQ_DATA $formatted_string $LOGIN $PASS
    echo $post_data

    local url=$URL'/'$TASK
    echo $url
    sleep 1

    local tmpfile=$(mktemp)
    # Run curl, writing the body to a temporary file and the status code to stdout
    local status_code=$(curl -s -w "%{http_code}" -o "$tmpfile" -d "$post_data" "$url")
    local body=$(cat "$tmpfile")
    rm "$tmpfile" # Clean up the temporary file

    echo "Status Code: $status_code"
    echo "Response Body: $body"

    if [ $status_code -eq 201 ]; then
        trimmed=$(echo "$body" | tr -d '"')
        tracking $trimmed $stop_state
        return $?
    else
        echo "Не получилось создать задачу на обработку $status_code"
    fi
}

function req_get() {
    local url=$1
    local cur_state=$2

    local tmpfile2=$(mktemp)
    # Run curl, writing the body to a temporary file and the status code to stdout
    local status_code=$(curl -s -w "%{http_code}" -o "$tmpfile2" "$url")
    local body=$(cat "$tmpfile2")
    rm "$tmpfile2" # Clean up the temporary file
    #echo "Status Code: $status_code"
    if [ $status_code -eq 200 ]; then
        #echo "Response Body: $body"
        if [ ${#body} -eq 0 ]; then
            echo "Нет данных по запросу: $1"
            return -1
        fi
        state=$(echo $body | grep -oP '"state":.+?,' | grep -Po '\d+')
        #echo "state $state"
        if [ $state -ne $cur_state ]; then
            echo -n "-= "; echo -n "$(get_state_string $cur_state)"; echo " $1 =-"
            echo -n "ID:              "; echo $body | jq -r '.id'
            echo -n "InDir:           "; echo $body | jq -r '.in_dir'
            echo -n "OutDir:          "; echo $body | jq -r '.out_dir'
            echo -n "URLS:            "; echo $body | jq -r '.urls | join(", ")'
            echo -n "Cmd:             "; echo $body | jq -r '.cmd'
            echo -n "Args:            "; echo $body | jq -r '.args | join(", ")'
            echo -n "OutExt:          "; echo $body | jq -r '.out_ext'
            echo -n "Files:           "; echo $body | jq -r '.files | join(", ")'
            echo -n "State:           "; get_state_string $state
            echo -n "Msg:             "; echo $body | jq -r '.msg'
            echo -n "Ftp:             "; echo $body | jq -r '.ftp[0]'
        fi

        return $state

    fi

    return -1
}

function req_delete() {
    local task_id=$1
    local url=$URL'/'$TASK'/'$task_id

    echo "Delete $url"
    curl --location --request DELETE $url
}

function tracking() {
    local url=$URL'/'$TASK'/'$1
    local stop_state=$2
    local state=0
    echo "Отслеживание $url"; echo ""

    while true; do
        req_get $url $state
        state=$?
        # остановка по статуса 127(ошибка), 5(Завершено)
        # статус 255 - если нечего отслеживать
        # можно задать до какого статуса ждем stop_state
        if [[ $state -eq 127 || $state -eq 5 || $state -eq 255 || $state -eq $stop_state ]]; then
            echo "Отслеживание завершено $(get_state_string $state) $1"
            return $state
        fi
        sleep 1
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
                req_post "${#array[@]}" "${array[@]}" &
                tasks=$((tasks+1))
                array=()
            fi
            if [ $tasks -eq $TASK_COUNT ]; then
                sleep 1
                tasks=0
                echo "WAIT tasks"

                wait
            fi
        done < "$CSV"
    else
        echo "Error: File '$CSV' not found."
    fi
}

function test_download_exec_ftp() {
    echo "Нажмите Ctrl+C для выхода"
    while true; do
        read_file
        echo "Полный проход по файлам"
        sleep 5
    done
}

# входящий параметр state
# ждем этот состояние и отключаем задачу
function test_stoping() {
    local url=$1
    local stop_state=$2
    req_post 1 $url $stop_state
}

if [[ ${#LOGIN} -eq 0 || ${#PASS} -eq 0 ]]; then
    echo "Пустой параметр LOGIN $LOGIN PASS $PASS"
    return
fi

#tracking "f5526c0c457337e5835672cac83c535c87ef192c2046f922e8e89f85f70a7627"
#req_delete "f5526c0c457337e5835672cac83c535c87ef192c2046f922e8e89f85f70a7627"
#my_array=("apple" "banana" "cherry")
#req_post $my_array

#test scripts
test_stoping "https://storage007.noisypeak.com/mastervod/cs/CelebrityScene01032025.mp4" 1
#test_stoping "https://storage007.noisypeak.com/mastervod/cs/CelebrityScene01032025.mp4" 2
#test_stoping "https://storage007.noisypeak.com/mastervod/cs/CelebrityScene01032025.mp4" 3
#test_download_exec_ftp

echo "Все задачи завершены"

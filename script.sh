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
    else
        echo "Не получилось создать задачу на обработку"
    fi
}

function req_get() {
    while true; do
        url=$URL'/'$TASK'/'$1
        local tmpfile=$(mktemp)
        # Run curl, writing the body to a temporary file and the status code to stdout
        local status_code=$(curl -s -w "%{http_code}" -o "$tmpfile" "$url")

        local body=$(cat "$tmpfile")
        rm "$tmpfile" # Clean up the temporary file
        echo "Status Code: $status_code"

        local cur_state=0
        if [ $status_code -eq 200 ]; then
            echo "Response Body: $body"
            if [ ${#body} -eq 0 ]; then
                echo "Нет данных по ключу $1"
                return
            fi

            state=$(echo $body | grep -oP '"state":.+?,' | grep -Po '\d+')
            echo "state $state"
            if [ $state -ne $cur_state ]; then
                cur_state=$state
                echo "Смена состояния $cur_state key $1"
            fi
            if [[ $state -eq 127 || $state -eq 5 ]]; then
                echo "Отслеживание завершено $state key $1"
                return
            fi
            continue
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
#req_get "b1ddf2f681d20de50f0865050846c5edab196550e1bc3289e5e0c6ceabee4a24"

echo "Все задачи завершены"

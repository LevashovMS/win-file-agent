#!/bin/bash

# params
URL="http://91.220.62.199:8080/v1"
REQ_DATA="{\"in_dir\":\"C:\\\\Users\\\\Administrator\\\\Downloads\\\\InDir\",\"urls\":[%s],\"cmd\":\"ffmpeg.exe\",\"args\":[\"-i\",\"{input}\",\"-c:v\",\"libx264\",\"-b:v\",\"500k\",\"-c:a\",\"copy\",\"{output}\"],\"out_ext\":\"mp4\",\"ftp\":{\"addr\":\"storage007.noisypeak.com:21\",\"login\":\"winworkflow\",\"pass\":\"ish7eeQu\"}}"
CSV="CS100files.txt"
TASK_COUNT=3
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
    IFS=','
    printf -v formatted_string '"%s"' "${arr[*]// /|}"
    ( IFS=','; printf '"%s"' "${arr[*]// /|}" )
    #echo "${formatted_string}"

    #printf -v formatted_string $REQ_DATA $formatted_string
    echo "${formatted_string}"

    sleep 3
    #curl -i --location '$URL/$TASK' \
    #--header 'Content-Type: application/json' \
    #--data '$REQ_DATA'
}

function req_get() {
    curl -i --location '$URL/$TASK/$1' \
    --header 'Content-Type: application/json'
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

processBar()
{
    process=$1
    whole=$2
    printf "[%03d/%03d]\r" $process $whole
}

whole=100
process=0
while [ $process -lt $whole ]
do
    let process++
    processBar $process $whole
    sleep 0.1
done

printf "\n"
i=1
while true
do

    let i++
    echo "log-"+$i
    sleep 1
done

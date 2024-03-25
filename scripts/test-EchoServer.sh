docker build -t ubuntu_netcat ./scripts
make docker-compose-up-clients CLIENTS=0
docker run -d -t --name client --network tp0_testing_net ubuntu_netcat 
docker start client
docker cp ./scripts/test-EchoServer-Client.sh client:/
docker exec -it client sh  test-EchoServer-Client.sh
echo
echo
sleep 2
docker stop client
docker rm client
make docker-compose-down
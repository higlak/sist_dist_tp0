FROM ubuntu:latest

RUN apt-get update && apt-get install -y netcat

COPY test-EchoServer-Client.sh /test-EchoServer-Client.sh

RUN chmod +x /test-EchoServer-Client.sh

CMD ["/bin/sh"]
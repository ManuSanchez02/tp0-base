FROM ubuntu
RUN apt-get update && apt-get install -y netcat
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "echo ping | netcat server 12345" ]
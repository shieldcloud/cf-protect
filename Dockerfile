FROM shieldcloud/agent
RUN apt-get update \
 && apt install -y mysql-client

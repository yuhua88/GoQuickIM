FROM golang:1.18

WORKDIR /go/src/gochat

COPY . .

RUN apt-get update && apt-get install -y gcc automake autoconf make supervisor

#etcd redis mysql
RUN cd /tmp && wget https://github.com/etcd-io/etcd/releases/download/v3.4.3/etcd-v3.4.3-linux-amd64.tar.gz \
    && tar -zxvf etcd-v3.4.3-linux-amd64.tar.gz \
    && cp etcd-v3.4.3-linux-amd64/etcd /bin/ \
    && cp etcd-v3.4.3-linux-amd64/etcdctl /bin/ \
    && rm -rf etcd-v3.4.3-linux-amd64.tar.gz && rm -rf etcd-v3.4.3-linux-amd64

RUN cd /tmp && wget http://download.redis.io/releases/redis-5.0.9.tar.gz -O redis.tar.gz \
    && tar -zxvf redis.tar.gz \
    && cd redis-5.0.9 && make \
    && cp ./src/redis-server /bin/ \
    && cp ./src/redis-cli /bin/ \
    && rm -rf redis.tar.gz && rm -rf redis-5.0.9

RUN cd /tmp && wget https://dev.mysql.com/get/mysql-apt-config_0.8.17-1_all.deb \
    && dpkg -i mysql-apt-config_0.8.17-1_all.deb \
    && apt-get update \
    && apt-get install -y mysql-server \
    && rm -rf mysql-apt-config_0.8.17-1_all.deb

ENV RUN_MODE dev
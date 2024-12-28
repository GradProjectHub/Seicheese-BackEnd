# Go
FROM golang:1.23.2 AS go-base

# Port to expose
EXPOSE 1300

# Create User
ARG USERNAME=user
ARG GROUPNAME=user
ARG UID=1000
ARG GID=1000

RUN groupadd -g $GID $GROUPNAME && useradd -m -s /bin/bash  -u $UID -g $GID $USERNAME

# Set Home Environment
ENV HOME /home/${USERNAME}

# Set GOPATH
ENV GOPATH ${HOME}/go

# ディレクトリを作成し、権限を設定
RUN mkdir -p ${GOPATH}/pkg && \
    chown -R ${USERNAME}:${GROUPNAME} ${HOME}

# Go-dev
FROM go-base AS go-dev

# Workdir
WORKDIR /home/user/go/src/app

# Set permissons
RUN chown -R $USERNAME:$GROUPNAME /home/user/go

# Air Install
RUN go install github.com/air-verse/air@latest

# Goose Install
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# SQLBoiler Install
RUN go install github.com/volatiletech/sqlboiler/v4@latest

# SQLBoiler MySQL Drivers Install
RUN go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-mysql@latest

# PATH
ENV PATH $PATH:${GOPATH}/bin



# Database
FROM mysql:8.4.0 AS db-base

# Port to expose
EXPOSE 3306

# Copy to my.cnf
COPY ./mysql/my.cnf /etc/mysql/my.cnf

# Set Common Environments
ENV TZ=Asia/Tokyo
ENV MYSQL_DATABASE=SeiCheese


# Database-dev
FROM db-base AS db-dev

# Set Development Environments
ENV MYSQL_ROOT_PASSWORD=Wario-51



# Phpmyadmin
FROM phpmyadmin:5.2.1 AS admin

# Port to expose
EXPOSE 80

# Set Environments
ENV PMA_HOST=db
ENV PMA_USER=root
ENV PMA_PASSWORD=Wario-51

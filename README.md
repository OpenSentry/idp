# golang-idp-be
Golang Identity Provider Backend

# Getting started
First of all make sure docker is installed and ready to use.

Next, run the following commands:
```
$ git clone https://github.com/linuxdk/golang-idp-be.git
$ cd golang-idp-be
$ # This will build a docker image by getting all necessary requirements and compiling the go project.
$ docker build -t idp .
$ # When the image has been build, use the following docker command to start it up:
$ docker run -it -p 8080:8080 -v $(pwd):/go/src/idp idp
```

Note that the default settings is a development build, which can be used for automatic rebuilding of the go code with the help of https://github.com/pilu/fresh. Later on the environment variable `APP_ENV` will be used to define a production or development environment

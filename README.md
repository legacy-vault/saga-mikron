# SAGA MIKRON
Web Chat «SAGA MIKRON»

## Project Summary

'Saga Mikron' is a simple web chat written in Go programming language (also known as Golang). This chat consists of a server and a client. The client part is embedded into server part, so there is no need for a separate client. 

This chat has ultra light weight. The client uses simple HTML without any graphics. Even buttons are made of standard HTML objects. The whole client part weighs about 20 KiB web traffic. Message updates take several bytes, not even a KiB! The chat uses Go language (Golang) as a back-end and JavaScript as front-end. Message updates and user list updates are done via dynamic requests known as AJAX. Messages between the server and clients are transfered using JSON format. The chat supports users' names, passwords and messages in unicode UTF-8 encoding. The client complies with the modern HTML5 standard. 

The chat is so simple that it does not allow password changes. This is done to prolong the life of the storage device, where the database of users is stored. So, in other words, this web chat is great for SSD drives and other drives that use flash technology, which is known to have limited number of write/erase cycles. 

The client part has a network indicator which shows average "ping" to the server (time between request and reply). If the server suddenly goes offline, then the client will immediately see it. 

## Download & Installation

1. install golang from `https://golang.org`;
2. open terminal and `cd` <to_your_working_dir>
3. `git clone https://github.com/legacy-vault/saga-mikron`
4. `cd saga-mikron/build`
5. `./install.sh`
6. now you can run the program. 

## Usage

The program needs a database of users to operate. When you first run the program or wish to create a new database file, use '-cudf' option. This means "Create User Data File". In normal situation you don't need to use `-cudf`.

To show the list of available command line parameters, use `-h`.

The default settings are wise enough to make chat working and keep both network and server in good condition. Note that setting revisor intervals to values less than 1 (one second) will raise network traffic consumption and server's CPU load greatly, so, please, do not overoptimize :)


## License

 GNU GENERAL PUBLIC LICENSE Version 3

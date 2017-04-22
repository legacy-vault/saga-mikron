# saga-mikron
Web Chat «SAGA MIKRON»

## Project Summary

'Saga Mikron' is a simple web chat written in Go programming language (also known as Golang). This chat consists of a server and a client. The client part is embedded into server part, so there is no need for a separate client. 

This chat has ultra light weight. The client uses simple HTML without any graphics. Even buttons are made from standard HTML objects. The whole client part weights less than 15 KiB web traffic. Message updates take several bytes, not even a KiB! The chat uses Go language (Golang) as a back-end and JavaScript as front-end. Message updates and user list updates are done via dynamic requests known as AJAX. Messages between the server and clients are transfered using JSON format. The chat supports users' names, passwords and messages in unicode UTF-8 encoding. 

The chat is so simple that it does not allow password changes. This is done to prolong the life of the storage device, where the database of users is stored. So, in other words, this web chat is great for SSD drives and other drives that use flash technology, which is known to have limited number of write/erase cycles. 

## Download & Installation

1. open terminal
2. cd <to_your_working_dir>
3. git clone https://github.com/legacy-vault/saga-mikron
4. inside folder "saga-mikron/src" create folder "dat"
5. open .go files with your Go IDE
6. compile
7. have fun

## License

 GNU GENERAL PUBLIC LICENSE Version 3

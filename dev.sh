
export TIME_ZONE=Asia/Shanghai
ln -snf /usr/share/zoneinfo/$TIME_ZONE /etc/localtime && echo $TIME_ZONE > /etc/timezone


export MYSQL_ADDR=mysql_1
export MYSQL_PORT=3306
export MYSQL_DATABASE=book_management
export MYSQL_USER=root
export MYSQL_PASSWORD=root



./book_management 


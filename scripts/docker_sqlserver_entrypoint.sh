# SQL Server can take a while to start, this scripts waits until it's ready and then creates a database
for i in {1..90};
do
    /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P Password123 -d master -q "IF DB_ID('test_db') IS NULL CREATE DATABASE test_db"
    if [ $? -eq 0 ]
    then
        echo "setup complete"
        break
    else
        echo "not ready yet..."
        sleep 1
    fi
done
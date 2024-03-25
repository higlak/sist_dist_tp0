apt-get update
apt-get install -y netcat
echo "
Netcat up to date

Mandando al server el mensaje \"Hola, servidor\". Se recibe: 

"
echo "Hola, servidor" | nc server 12345
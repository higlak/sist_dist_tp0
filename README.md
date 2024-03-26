#### EJ 1:
Se definió un script de bash "scripts/docker-compose-client-generator" que escribe en un nuevo archivo las definiciones del docker compose. El docker-compose-dev.yaml final tendrá la definición del servidor al inicio y la red al final. En el medio se definirán n clientes, cada uno con el nombre cliente{i} y  con la variable de entorno CLI_ID=i. Luego, cuando aparecieron los volúmenes se agregaron estos a las definiciones en el .yaml. Entonces al servidor se le agrega la definición de un volumen de configuración, y al cliente uno de configuración. También se agregó un comando extra en el makefile docker-compose-up-clients que antes de hacer el comprise up llama al script con el valor que tengo la variable de entorno CLIENTS. Si esta no esta inicializada por default instancia 2

#### Ej2:
Para evitar tener que hacer un nuevo build de las imágenes de docker cuando hay algún cambio en el config, ya sea del cliente o del server se agregaron dos volúmenes de tipo bind, uno para el servidor, y uno para los clientes. Se hace el bind entre  una carpeta del docker y una carpeta de la máquina local, permitiendo que la modificación de los archivos dentro de la carpeta desde la máquina local se vea reflejada automáticamente en el container. De esta manera al hacer un cambio en el config, al estar en un volumen, no se hará una nueva imagen, simplemente se utilizará la última y al iniciar se verán los cambios reflejados en el config.

#### Ej3:
Para correr el test se debe correr el script test-EchoServer.sh . Este script buildeara tanto el container del servidor e iniciara la red, pero sin levantar un cliente. Luego se hace un build con el dockerfile de la carpeta script, esta imagen será la del container cliente que se comunique con el servidor. Una vez levantado el cliente se lo inicia y se lo manda a ejecutar el script teste-EchoServer-Client.sh, que instala netcat en el container y manda un mensaje al echoserver. finalmente se bajan todos los containers.

#### Ej4:
A continuación se explica cómo se implementó el gracefull finish tanto en el servidor como en el cliente.
##### Servidor:
En el servidor defini un handler para la señal SIG_TERM que cierra los sockets, tanto el que escucha nuevas conexiones, como el que se utiliza para comunicar con un cliente ya establecido. Cuando se levanta una SIG_TERM, se corta la ejecución normal del código y se ejecuta este handler, antes de seguir con la ejecución normal. Esto hace que el código que se venía ejecutando con sockets abiertos, ahora los tenga cerrados. Lanzando asi excepciones que son catcheadas por los try, que cortan el ciclo de ejecución infinito del servidor, haciendo que termine naturalmente el programa.
##### Cliente:
El código del cliente consta de un loop donde manda mensajes y espera la respuesta del echoserver. Para hacer el gracefull finish, lo que hice fue crear un channel por el cual voy a recibir la SIG_TERM. Esto me permite que al finalizar cada iteración pueda intentar recibir del channel para ver si se debe finalizar. Si ya paso LoopPeriod segundos, se continúa con el envío y recepción del próximo mensaje. Ahora, esto trae un problema, que es el de operaciones bloqueantes en el loop (como un recv) que hacen que si nuestro programa estaba bloqueado cuando se envía la SIG_TERM no nos enteremos, y nuestro programa será frenado con un SIG_KILL. Para evitar esto, el recv ahora se hace desde una go routine que manda por un chanel lo que recibe. De esta manera se pueden hacer todas las operaciones no bloqueantes al inicio del loop, y luego en un select, quedarse esperando por el primero de 4 eventos. Ya sea terminar el loop porque ocurrio un timeout de LoopLapse, terminar el loop si se recibe un SIG_TERM, o recibir el mensaje y seguir con la siguiente iteración después de LoopPeriod segundos. Este select garantiza que la señal será escuchada independientemente de que operación bloqueante se esté realizando.

#### Ej5:
##### Protocolo
Para resolver el ejercicio se plantea el siguiente protocolo de comunicacion. Luego de hacer una conexion tcp, el cliente, mandara una bet al servidor, y este le respondera con un ack. El mensaje bet que envia el cliente esta compuesto por un header que contiene los campos:  
- agency(2 bytes): Es un u16 en big endian order que representa el numero de la agencia
- date(4 bytes): El primer bytes indica el dia, el segundo el mes, y los ultimos dos se interpretan como un u16 en big endian order que representa 
- dni(4 bytes): Es un u32 en big endian order que representa el numero de dni
- lottery_number(4 bytes): Es un u32 en big endian order que representa el numero de loteria
- name_len(1 bytes): Es un u32 en big endian order que representa la cantidad de bytes que usan los nombres
Luego como paiload de mensaje se enviara el nombre y apellido. Se enviaran ambos dentro del mismo campo separados por un ';'. Los clientes entonces tendran como maximo 255 bytes, debido a que name_len tiene un bytes, de los cuales uno es usado para el separador. Entonces se define que el nombre y el apellido pueden tener como maximo 127 bytes cada uno, si un nombre o apellido tiene mas caracteres se tomo la desicion de truncarlo.

![bet_fields.png](./fotos/bet_flieds.png)

Este protocolo pone varias restricciones sobre la longitud o cantidad de cosas, por ejemplo la longitud de nombre o cantidad de agencias, pero estas asumo son lo suficientemente holgadas, y se podrian agrandar simplemente cambiando el tipo de dato.
Entonces luego de que el cliente parsee y envie los datos, el servidor recibira primero un header de tama;o fijo 15, y luego lo usara para obtener cuantos bytes leer de los nombres. Una vez recibida y almacenada la bet el servidor responder con un ack. En cuanto al protocolo respecta este es un mensaje de un byte, sin importar su contenido. El servidor en la practica enviara siempre el byte 255. El cliente verificara la llegada de ese byte para verificar el almacenamiento de la bet.
 
##### Short Read y short write
Para evitar el short read y el short write, implemente las funciones send_all y recv_exactly. Send all envia todo un array de bytes, y recv_exactly recive exactamente una cantidad de bytes. Esto se logra enviando o recibiendo en principio todos lo bytes, si se enviaron o recibieron menos, entonces se repite la operacion con los bytes restantes hasta haber enviado o recibido todos los bytes.

##### Aclaracion
Decidi mantener la logica anterior donde los clientes enviaban multiples veces el mismo mensaje cada LoopPeriod dentro del loop, siempre que no exedan el LoopLapse. 

#### Ej6
##### Volume Data
Primero para conseguir que los containers clientes tuvieran acceso a los archivos dentro de la carpeta data, cree un nuevo volumen data.
##### Protocolo
Para este ejercicio se pedía enviar múltiples bets en un batch, antes de que el server respondiera con su ack. Para hacer esto se define un nuevo mensaje en el protocolo, mensaje batch, que contiene en el header un byte que indica la cantidad de bets que contiene el batch. Mientras que el payload es un conjunto de bets como se definió en el punto anterior.   
<img src="./fotos/bet_batch.png" alt="drawing" width="200"/>   
De esta manera luego de establecer la conexión tcp, el cliente leerá del csv de a batches y lo enviará al server quien le responderá con un ack siempre que haya podido almacenarlo. Una vez enviadas todas las bets, el cliente enviará un batch con 0 bets, para comunicarle al servidor que ya no tiene más batches a mandar. 
Esta solución tiene un problema que es que se procesa un cliente a la vez. Si primero se procesa un cliente con muchas bets, como el cliente1, y luego llega un cliente con muchas menos, como el 5, este último deberá esperar a que el cliente1 finalice de enviar todas sus bets, antes de poder empezar a procesar las suyas. Es decir que tenemos un problema de convoy. La otra solución, hubiera sido cerrar la conexión tcp cada vez que se recibe un batch, de esta manera se irá intercalando a qué cliente se procesa. Esta segunda solución tampoco es perfecta, ya que se crearía y cerraría una conexión tcp por batch, en vez de una por cliente. Debido a que en el siguiente punto, se debe esperar a que todos los clientes terminen antes de continuar la ejecución, concluyó que la primera solución es más adecuada, ya que en ese contexto el problema del convoy no ocasiona efecto alguno.
##### Bets por batch
Debido a los tamaños fijos y máximos de las bets, se puede asegurar una cota máxima para el tamaño de las bet. El header mide 15 bytes, y en el peor de los casos hay 255 bytes usados en el nombre y apellido, por lo que en el peor de los casos el mensaje bet mide 270 bytes. Es decir que si queremos mandar menos de 8kb por batch, cada batch deberá tener 8000/270 bytes, es decir 29,6 bets. Para poner un número redondo fijo el tamaño default de las batch en 25. 
El tamaño máximo de un batch es configurable desde el configuración del cliente.

#### Ej7
##### Identificar finalizacion de clientes
Para obtener los ganadores primero el servidor debe enterarse que todos los clientes han finalizado de enviar sus bets. Esto ya se hizo en el punto anterior, cada cliente cuando finalize de mandar sus bets enviara un batch vacio para indicar que no tiene mas bets a enviar.
##### Winners
Para calcular los ganadores el servidor utiliza la funcion load bets para ir leyendo del archivo una bet a la vez y fijandose con isWinner si gano. Una vez obtenido los ganadores, envia a cada cliente sus ganadores. Esto lo hace enviando primero la cantidad de ganadores seguido de todos los dni. Los clientes lo reciben y loguean acordemente.   
<img src="./fotos/bet_batch.png" alt="drawing" width="200"/>   
Si bien el servidor solo manda los dni, si el cliente quisiera saber mas informacion sobre el ganador la podria obtener desde su archivo de bets.

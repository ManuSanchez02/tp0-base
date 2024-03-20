# TP0: Docker + Comunicaciones + Concurrencia

## Ejercicio 1
Para ejecutar el ejercicio 1, simplemente se debe ejecutar el comando

```bash
make docker-compose-up
```

Esto ejecutara el docker-compose y creara los contenedores respectivos. Para verificar que haya un cliente nuevo, podemos ver los logs usando:

```bash
make docker-compose-logs
```

### Ejercicio 1.1
En el directorio `ejercicio_1` hay un script de Python llamado `main.py`. Este script genera un archivo `docker-compose-ej-1.yaml`. Para correrlo, se usa de la siguiente manera:

```bash
python3 main.py <CANTIDAD_DE_CLIENTES>
```

Esto genera un archivo de configuracion docker compose con la cantidad de clientes ingresada. Finalmente para ejecutar dicho archivo y corroborar que la cantidad de clientes sea correcta, se pueden repetir los mismos comandos del ejercicio 1, pero con una pequeña diferencia en el nombre:

```bash
make docker-compose-up-ej-1
make docker-compose-logs-ej-1
```

De esta forma, se levantan los contenedores indicados en el archivo de configuracion `docker-compose-ej-1.yaml`. Para detener estos contenedores, se puede usar el comando `make docker-compose-down-ej-1`.


## Ejercicio 2
Para verificar el correcto funcionamiento del ejercicio 2, basta con ejecutar alguno de los archivos de configuracion `docker-compose.yaml` para levantar los contenedores. Verificar el correcto funcionamiento de los mismos mediante los logs o `docker ps -a` (verificando que el codigo de salida sea 0).

Luego, es posible modificar los archivos de configuracion `client/config.yaml` o `server/config.ini`. Por ejemplo, si el servidor sigue corriendo con la configuracion inicial, y se vuelve a ejecutar algun contenedor de cliente pero habiendo cambiado la configuracion en `client/config.yaml` por:
```yaml
# cliente (configuracion nueva)
server:
  address: "server:12344"
loop:
  lapse: "0m20s"
  period: "5s"
log:
  level: "info"
  ```

```ini
# servidor (configuracion inicial)
[DEFAULT]
SERVER_PORT = 12345
SERVER_IP = server
SERVER_LISTEN_BACKLOG = 5
LOGGING_LEVEL = INFO
```

Al no coincidir los puertos, el cliente deberia arrojar un error. Para probarlo, se puede levantar el contenedor del cliente con el comando:
```bash
docker start <ID_CONTENEDOR>
```

Luego, solo resta verificar los logs o el codigo de salida (mediante `docker ps -a`) del cliente para ver si los cambios surtieron efecto.


## Ejercicio 3
En la carpeta de `ejercicio_3` se encuentra todo lo necesario para verificar si el servidor esta ejecutandose. El primer paso es ejecutar el script `build_image.sh`, el cual construye la imagen que se usara para realizar la verificacion. 

Una vez ejecutado dicho script, tan solo resta ejecutar `run_container.sh`. Este script levanta el contenedor, el cual, usando netcat, envia el mensaje `"ping"` al servidor. 

El servidor, al ser un EchoServer, responde con el mismo mensaje. Si la respuesta recibida es igual a `"ping"`, entonces la verificacion ha tenido exito. En caso contrario, se imprimira un mensaje indicando que hubo un error.

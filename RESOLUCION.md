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
OUTPUT=$(sudo docker run --rm --network tp0_testing_net --name ping_server ping:latest)

if [ "$OUTPUT" == "ping" ]; then
    echo "El servidor respondio correctamente"
else
    echo "No se obtuvo respuesta del servidor"
fi

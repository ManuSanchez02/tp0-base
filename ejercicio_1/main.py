from argparse import ArgumentParser
import yaml


def create_config(clients):
    config = {
        "version": "1.0",
        "name": "ejercicio_1_1",
        "services": {
                "server": {
                    "container_name": "server",
                    "image": "server:latest",
                    "entrypoint": "python3 /main.py",
                    "environment": [
                        "PYTHONUNBUFFERED=1",
                        "LOGGING_LEVEL=DEBUG"
                    ],
                    "networks": [
                        "testing_net"
                    ],
                    "volumes": [
                        "./server/config.ini:/config.ini"
                    ]
                }
        },
        "networks": {
            "testing_net": {
                "ipam": {
                    "driver": "default",
                    "config": [
                        {
                            "subnet": "172.25.125.0/24"
                        }
                    ]
                }
            }
        },
    }

    for i in range(clients):
        client_id = i+1
        client_config = {
            "container_name": f"client{client_id}",
            "image": "client:latest",
            "entrypoint": "/client",
            "environment": [
                f"CLI_ID={client_id}",
                "CLI_LOG_LEVEL=DEBUG",
                f"CLI_SOURCE=agency-{client_id}.csv"
            ],
            "networks": [
                "testing_net"
            ],
            "volumes": [
                "./client/config.yaml:/config.yaml",
                "./.data:/.data"
            ],
            "depends_on": [
                "server"
            ]
        }
        config["services"][f"client{client_id}"] = client_config

    return config


def get_client_number():
    parser = ArgumentParser(
        prog='Docker compose config generator',
        description='Generate n clients')
    parser.add_argument('client_number', type=int, help="Number of clients")
    args = parser.parse_args()
    return args.client_number


def main():
    client_number = get_client_number()
    file_data = create_config(client_number)
    with open("docker-compose-ej-1.yaml", "w") as file:
        yaml.dump(file_data, file, default_flow_style=False, sort_keys=False)


if __name__ == "__main__":
    main()

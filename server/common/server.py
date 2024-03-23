import signal
import socket
import logging

from common.utils import Bet, store_bets

CONFIRMATION = b'OK'

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._client_sockets = set()

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        signal.signal(signal.SIGTERM, self.__graceful_shutdown)

        while True:
            try:
                client_sock = self.__accept_new_connection()
            except OSError:
                logging.info("action: graceful_shutdown | result: success")
                return
            
            self.__handle_client_connection(client_sock)

    def __read_msg(self, client_sock: socket.socket) -> str:
        packet_len = int.from_bytes(receive_exact(client_sock, 1), 'big', signed=False)
        msg = receive_exact(client_sock, packet_len).decode('utf-8')        
        addr = client_sock.getpeername()
        logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')

        return msg
    
    def __send_confirmation(self, client_sock: socket.socket):
        send(client_sock, CONFIRMATION)
        addr = client_sock.getpeername()
        logging.info(f'action: send_confirmation | result: success | ip: {addr[0]}')

    def __handle_client_connection(self, client_sock: socket.socket):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = self.__read_msg(client_sock)
            bet = Bet.from_str(msg)
            store_bets([bet])
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')
            self.__send_confirmation(client_sock)

        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        except ValueError as e:
            logging.error(f"action: apuesta_almacenada | result: fail | error: {e}")
        finally:
            client_sock.close()
            self._client_sockets.discard(client_sock)

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        self._client_sockets.add(c)
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def __graceful_shutdown(self, _signum, _frame):
        logging.info("action: graceful_shutdown | result: in_progress")
        logging.info("action: close_server_socket | result: in_progress")
        self._server_socket.close()
        logging.info("action: close_server_socket | result: success")
        logging.info("action: close_client_sockets | result: in_progress")
        for client_sock in self._client_sockets:
            client_sock.close()
        logging.info("action: close_client_sockets | result: success")
        
def send(client_sock: socket.socket, message: bytes):
    sent_data = client_sock.send(message)
    while sent_data < len(message):
        sent_data += client_sock.send(message[sent_data:])

def receive_exact(client_sock: socket.socket, length: int) -> bytes:
    data = client_sock.recv(length)
    while len(data) < length:
        data += client_sock.recv(length - len(data))
    return data
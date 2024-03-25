from enum import Enum
import signal
import socket
import logging
from typing import Optional

from common.utils import Bet, has_won, load_bets, store_bets

CONFIRMATION = b'OK'
REQUIRED_AGENCIES = 5

class MessageType(Enum):
    BET = 0
    END = 1
    WINNERS = 2
    WINNER = 3
    

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._client_sockets = set()
        self.notifications = {}

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
        
        
    def __read_msg(self, client_sock: socket.socket) -> tuple[Optional[bytes], MessageType]:
        message_type = receive_exact(client_sock, 1)
        message_type = int.from_bytes(message_type, 'big', signed=False)
        message_type = MessageType(message_type)
        if message_type == MessageType.END:
            return None, message_type
        elif message_type == MessageType.WINNERS:
            return None, message_type
        elif message_type == MessageType.BET:
            return self.__receive_batch(client_sock), message_type
        
        raise ValueError(f"Invalid message type: {message_type}")

    def __receive_batch(self, client_sock: socket.socket) -> bytes:
        batch_len_bytes = receive_exact(client_sock, 4)
        addr = client_sock.getpeername()
        batch_len = int.from_bytes(batch_len_bytes, 'big', signed=False)
        logging.debug(f'action: receive_batch | result: in_progress | ip: {addr[0]} | batch_len: {batch_len}')
        batch = receive_exact(client_sock, batch_len)          
        logging.debug(f'action: receive_batch | result: success | ip: {addr[0]}')

        return batch
    
    def __process_batch(self, batch_bytes: bytes, client_sock: socket.socket, client_id: str):
        bets = parse_bets(batch_bytes, client_id)
        store_bets(bets)
        logging.info(f'action: apuestas_almacenadas | result: success | client_id: {client_id} | apuestas: {bets}')
        self.__send_confirmation(client_sock)
    
    def __send_confirmation(self, client_sock: socket.socket):
        send(client_sock, CONFIRMATION)
        addr = client_sock.getpeername()
        logging.debug(f'action: send_confirmation | result: success | ip: {addr[0]}')

    def __get_client_id(self, client_sock: socket.socket) -> str:
        id = ""
        while read := client_sock.recv(1):
            if read == b'\n':
                return id
            id += read.decode('utf-8')

    def __handle_client_connection(self, client_sock: socket.socket):
        """
        Read batches from client socket and process them. If an empty
        message is received, the client socket will be closed.

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            client_id = self.__get_client_id(client_sock)

            while True:
                msg, msg_type = self.__read_msg(client_sock)
                if msg_type == MessageType.END:
                    self.notifications[client_id] = True
                    break
                elif msg_type == MessageType.WINNERS:
                    if self.__all_notifications_received():
                        self.send_winners(client_sock, client_id)
                    break
                elif msg_type == MessageType.BET:
                    self.notifications[client_id] = False
                    self.__process_batch(msg, client_sock, client_id)

        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        except ValueError as e:
            logging.error(f"action: apuestas_almacenadas | result: fail | error: {e}")
        except EOFError:
            logging.info(f"action: client_disconnected | result: success | client_id: {client_id}")
        finally:
            client_sock.close()
            self._client_sockets.discard(client_sock)

    def __all_notifications_received(self):
        return all(self.notifications.values()) and len(self.notifications) == REQUIRED_AGENCIES
    
    def send_winners(self, client_sock: socket.socket, client_id: str):
        for bet in load_bets():
            if bet.agency == int(client_id) and has_won(bet):
                msg = bet.serialize().encode('utf-8')
                length = len(msg).to_bytes(1, 'big')
                bet_message = MessageType.WINNER.value.to_bytes(1, 'big')
                msg = bet_message + length + msg
                send(client_sock, msg)
                logging.debug(f"action: send_winners | result: in_progress | client_id: {client_id} | bet: {bet}")

        end_message = MessageType.END.value.to_bytes(1, 'big')
        send(client_sock, end_message)
        logging.info(f"action: send_winners | result: success | client_id: {client_id}")

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
        read_data = client_sock.recv(length - len(data))
        if not read_data:
            raise EOFError("EOF reached while reading data")
        data += read_data
    return data

def parse_bets(batch: bytes, agency_id: str):
    bets = []
    i = 0
    while i < len(batch): 
        packet_len = batch[i]
        bet_info = f"{agency_id};{batch[i+1:i+packet_len+1].decode('utf-8')}"
        current_bet = Bet.from_str(bet_info)
        bets.append(current_bet)
        i += packet_len + 1
    return bets

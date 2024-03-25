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
        
        
    def __read_msg(self, client_sock: socket.socket) -> bytes:
        batch_len_bytes = client_sock.recv(4)
        if not batch_len_bytes:
            return b''
        
        if len(batch_len_bytes) != 4:
            raise OSError("Error while reading batch length")
        
        addr = client_sock.getpeername()
        batch_len = int.from_bytes(batch_len_bytes, 'big', signed=False)
        logging.info(f'action: receive_batch | result: in_progress | ip: {addr[0]} | batch_len: {batch_len}')

        batch = client_sock.recv(batch_len)
        while len(batch) < batch_len:
            batch += client_sock.recv(batch_len - len(batch))
            logging.info(f"action: receive_batch | result: in_progress | read_len: {len(batch)}")
          
        logging.info(f'action: receive_batch | result: success | ip: {addr[0]}')

        return batch
    
    def __send_confirmation(self, client_sock: socket.socket):
        sent_data = client_sock.send(CONFIRMATION)
        while sent_data < len(CONFIRMATION):
            sent_data += client_sock.send(CONFIRMATION[sent_data:])
        
        addr = client_sock.getpeername()
        logging.info(f'action: send_confirmation | result: success | ip: {addr[0]}')

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
            best_processed = 0

            while True:
                msg_bytes = self.__read_msg(client_sock)
                if not msg_bytes:
                    logging.info("action: client_disconnected | result: success")
                    break

                bets = process_batch(msg_bytes, client_id)
                store_bets(bets)
                best_processed += len(bets)
                logging.info(f'action: apuestas_almacenadas | result: success | apuestas: {bets}')
                self.__send_confirmation(client_sock)
            logging.info(f"action: client_disconnected | result: success | bets_processed: {best_processed}")
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        except ValueError as e:
            logging.error(f"action: apuestas_almacenadas | result: fail | error: {e}")
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
        

def process_batch(batch: bytes, agency_id: str):
    bets = []
    i = 0
    while i < len(batch): 
        packet_len = batch[i]
        bet_info = f"{agency_id};{batch[i+1:i+packet_len+1].decode('utf-8')}"
        current_bet = Bet.from_str(bet_info)
        bets.append(current_bet)
        i += packet_len + 1
    return bets
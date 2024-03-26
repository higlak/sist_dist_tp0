import socket
import logging
import signal
from common.utils import NAME_LEN_BYTE_POSITION, HEADER_LEN, Bet, store_bets, recv_exactly, send_all

STORED_BET_MSG = bytearray([0xff])

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.client_socket = None
        signal.signal(signal.SIGTERM, self.handle_SIG_TERM)

    def run(self):
        while True:
            try:
                self.__accept_new_connection()
                self.__handle_client_connection(self.client_socket)
            except:
                break
        self.close_sockets()

    def close_sockets(self):
        self._server_socket.close()
        self.client_socket.close()

    def handle_SIG_TERM(self, _signum, _frame):
        self.close_sockets()

    def __handle_client_connection(self, client_sock):
        try:
            # TODO: Modify the receive to avoid short-reads
            msg_header = recv_exactly(self.client_socket, HEADER_LEN)
            if msg_header == None:
                return
            names = recv_exactly(self.client_socket, msg_header[NAME_LEN_BYTE_POSITION])
            if names == None:
                return
            bet = Bet.from_bytes(msg_header + names)
            store_bets([bet])
            logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')

            send_all(self.client_socket, STORED_BET_MSG)

        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            self.client_socket.close()


    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        self.client_socket = c

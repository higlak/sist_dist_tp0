import socket
import logging
import signal
from common.utils import BET_BATCH_HEADER_LEN, NAME_LEN_BYTE_POSITION, BET_HEADER_LEN, Bet, store_bets, recv_exactly, send_all

STORED_BET_MSG = bytearray([0xff])
AGENCIES = 5

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.client_sockets = []
        signal.signal(signal.SIGTERM, self.handle_SIG_TERM)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        while len(self.client_sockets) < AGENCIES:
            try:
                self.__accept_new_connection()
            except:
                return self.close_sockets()
            self.__handle_client_connection(self.client_sockets[len(self.client_sockets)-1])

        logging.debug("Cerrando socket")
        self.close_sockets()

    def close_sockets(self):
        self._server_socket.close()
        for socket in self.client_sockets:
            socket.close()

    def handle_SIG_TERM(self, _signum, _frame):
        self.close_sockets()

    def __handle_client_connection(self, client_socket):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            ammount_of_bets = 0
            while True:
                bet_batch = self.recv_bet_batch(client_socket)
                if bet_batch == None:
                    return
                if len(bet_batch) == 0:
                    break
                store_bets(bet_batch)
                ammount_of_bets += len(bet_batch)
                send_all(client_socket, STORED_BET_MSG)
                #logging.info(f'action: batch_almacenado | result: success | cantidad de bets: {len(bet_batch)}')
            logging.info(f'action: stored batches | result: success | cantidad de bets: {ammount_of_bets}')
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")

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
        self.client_sockets.append(c)
        #return c

    def recv_bet_header(self, client_socket):
        return recv_exactly(client_socket, BET_HEADER_LEN)

    def recv_bet(self, client_socket):
        bet_header = self.recv_bet_header(client_socket)
        if bet_header == None:
            return None
        names = recv_exactly(client_socket, bet_header[NAME_LEN_BYTE_POSITION])
        if names == None:
            return None
        bet = Bet.from_bytes(bet_header + names)
        return bet

    def recv_bet_batch(self, client_socket):
        amount_of_bets = self.recv_bet_batch_header(client_socket)
        if amount_of_bets == None:
            return None
        bet_batch = []
        for _i in range(amount_of_bets):
            bet = self.recv_bet(client_socket)
            if bet == None:
                return None
            bet_batch.append(bet)
        return bet_batch

    #Receives one byte from the client socket which represents the amount of bets that
    #are going to be sent by the client
    def recv_bet_batch_header(self, client_socket):
        amount_of_bets = recv_exactly(client_socket, BET_BATCH_HEADER_LEN)
        if amount_of_bets != None:
            amount_of_bets = amount_of_bets[0]
        return amount_of_bets

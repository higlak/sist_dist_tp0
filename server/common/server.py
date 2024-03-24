import socket
import logging
import signal
import threading
from common.threard_coordination import *
from common.utils import *

STORED_BET_MSG = bytearray([0xff])
AGENCIES = 5

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.client_sockets = []
        self.threads = []
        signal.signal(signal.SIGTERM, self.handle_SIG_TERM)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        winner_sender, winner_receivers = create_winner_comunicators(AGENCIES)
        safe_bet_writer = SafeBetWriter()
        while len(self.client_sockets) < AGENCIES:
            try:
                self.__accept_new_connection()
                thread = threading.Thread(target=self.__handle_client_connection, args=(
                    self.client_sockets[len(self.client_sockets) -1],
                    safe_bet_writer,
                    winner_receivers[len(self.client_sockets) -1]
                    ))
                thread.start()
                self.threads.append(thread)
            except:
                return self.close_sockets()
            #self.__handle_client_connection(self.client_sockets[len(self.client_sockets)-1])

        logging.info("action: sorteo | result: success")
        winner_sender.wait_for_agencies()
        logging.debug("Termine de esperar a las agencias")
        winners = get_winners()
        winner_sender.send_winners(winners)
        
        logging.debug("Cerrando socket")
        self.join_threads()
        self.close_sockets()

    def send_winners(self, socket, winner_receiver: WinnerReceiver):
        agency_bytes = recv_exactly(socket, AGENCY_BYTES)
        if agency_bytes == None:
            return
        agency = byte_array_to_big_endian_integer(agency_bytes)
        winners = winner_receiver.recv_all_winners_from_agency(agency)

        data = integer_to_big_endian_byte_array(len(winners), AMOUNT_OF_WINNERS_BYTES)
        for winner_dni in winners:
            logging.debug(f"action: winner | result: success | agency: {agency} | dni: {winner_dni}")
            data += integer_to_big_endian_byte_array(winner_dni, DOCUMENT_BYTES)
        send_all(socket, data)

    def join_threads(self):
        for thread in self.threads:
            thread.join()

    def close_sockets(self):
        self._server_socket.close()
        for socket in self.client_sockets:
            socket.close()

    def handle_SIG_TERM(self, _signum, _frame):
        self.close_sockets()

    # Receives and stores all bet batches. If all are stored it returns the amount of bets.
    # On failure None is returned
    def recv_all_client_batches(self, client_socket, safe_bet_writer: SafeBetWriter):
        amount_of_bets = 0
        while True:
            bet_batch = self.recv_bet_batch(client_socket)
            if bet_batch == None:
                return None
            if len(bet_batch) == 0:
                break
            safe_bet_writer.store_bets(bet_batch)
            amount_of_bets += len(bet_batch)
            send_all(client_socket, STORED_BET_MSG)
            #logging.info(f'action: batch_almacenado | result: success | cantidad de bets: {len(bet_batch)}')
        return amount_of_bets

    def __handle_client_connection(self, client_socket, safe_bet_writer: SafeBetWriter, winner_receiver: WinnerReceiver):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            amount_of_bets = self.recv_all_client_batches(client_socket, safe_bet_writer)
            if amount_of_bets == None:
                return
            logging.info(f'action: stored batches | result: success | thread: {threading.current_thread().ident} |cantidad de bets: {amount_of_bets}')
            self.send_winners(client_socket, winner_receiver)
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | thread: {threading.current_thread().ident} | error: {e}")
        finally:
            winner_receiver.finished()

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

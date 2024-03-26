from multiprocessing import Lock, Pipe, connection

from common.utils import Bet, store_bets

class SafeBetWriter():
    def __init__(self):
        self.lock = Lock()
    
    def store_bets(self, bets: list[Bet]):
        with self.lock:
            store_bets(bets)

class WinnerReceiver():
    def __init__(self, conn: connection.Connection):
        self.conn = conn

    def recv_all_winners_from_agency(self, agency: int):
        self.conn.send(agency)
        winners = []
        while True:
            winner = self.conn.recv()
            if winner == None: 
                break
            winners.append(winner)
        return winners
    
    def finished(self):
        self.conn.send(None)

class WinnerSender():
    def __init__(self, conns: list[connection.Connection]):
        self.conns = conns
        self.agencies = []

    def wait_for_agencies(self):
        for conn in self.conns:
            agency = conn.recv()
            self.agencies.append(agency)

    def send_winners(self, winners):
        for agency, conn in zip(self.agencies, self.conns):
            #skips any threads that finished before getting to send the agency
            if agency == None:
                continue
            for winner in winners.get(agency, []):
                conn.send(winner)
            conn.send(None)

def create_winner_comunicators(n):
    child_comunicators = []
    winner_receivers = []
    for _i in range(n):
        parent_conn, child_conn = Pipe()
        winner_receivers.append(WinnerReceiver(child_conn))
        child_comunicators.append(parent_conn)
    
    return WinnerSender(child_comunicators), winner_receivers
    

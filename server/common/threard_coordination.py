import logging
import signal
import queue
from threading import Lock

from common.utils import Bet, store_bets

class SafeBetWriter():
    def __init__(self):
        self.lock = Lock()
    
    def store_bets(self, bets: list[Bet]):
        with self.lock:
            store_bets(bets)

class WinnerReceiver():
    def __init__(self, agency_finished_queue: queue.Queue, winners_queueu: queue.Queue):
        self.agency_finished_queue = agency_finished_queue
        self.winners_queueu = winners_queueu

    def recv_all_winners_from_agency(self, agency: int):
        self.agency_finished_queue.put(agency)
        winners = []
        while True:
            winner = self.winners_queueu.get()
            if winner == None: 
                break
            winners.append(winner)
        return winners
    
    def finished(self):
        self.agency_finished_queue.put(None)

class WinnerSender():
    def __init__(self, agency_finished_queues: list[queue.Queue], winners_queueus: list[queue.Queue]):
        self.agency_finished_queues = agency_finished_queues
        self.winners_queueus = winners_queueus
        self.agencies = []

    def wait_for_agencies(self):
        for agency_finished_queue in self.agency_finished_queues:
            agency = agency_finished_queue.get()
            self.agencies.append(agency)

    def send_winners(self, winners):
        for agency, winners_queue in zip(self.agencies, self.winners_queueus):
            #skips any threads that finished before getting to send the agency
            if agency == None:
                continue
            for winner in winners.get(agency, []):
                winners_queue.put(winner)
            winners_queue.put(None)

def create_winner_comunicators(n):
    agency_finished_queues = []
    winners_queueus = []
    winner_receivers = []
    for _i in range(n):
        agency_finished_queue = queue.Queue()
        winners_queueu = queue.Queue()
        winner_receiver = WinnerReceiver(agency_finished_queue, winners_queueu)
        agency_finished_queues.append(agency_finished_queue)
        winners_queueus.append(winners_queueu)
        winner_receivers.append(winner_receiver)
    
    return WinnerSender(agency_finished_queues,winners_queueus), winner_receivers
    

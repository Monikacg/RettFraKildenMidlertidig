package network

import (
	"fmt"
	"strconv"
	"time"

	"sort"

	. "../definitions"
	"./bcast"
	"./peers"
)

// All functions used by this file (found in the peers, localip, conn and bcast folders)
// were made by github.com/klasbo.

// Note on packet loss: Bør sette opp timeout på peers-modulen? hvis det går tapt der?
// eller vil det bare ødelegge? Test!


func messageSender(helloTx chan<- OverNetwork, messageSenderChan <-chan OverNetwork) {
	for {
		select {
		case msg := <-messageSenderChan:
			//fmt.Println("NW::messageSender: Sending this: ", msg)
			helloTx <- msg
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func backupSender(backupTx chan<- BackUp, backupSenderChan <-chan BackUp) {
	for {
		select {
		case bu := <-backupSenderChan:
			//fmt.Println("NW::backupSender: Sending this: ", bu)
			backupTx <- bu
			time.Sleep(100 * time.Millisecond)
		}
	}
}


// VIKTIGST Å TEST OM MSGACKTIMER SLÅR UT RIKTIG NÅR DEN BLIR RESATT SÅ OFTE.
func checksIncomingMessages(IDInput int, ackCurrentPeersChan <-chan CurrPeers, adminToAckChan <-chan Udp, helloRx <-chan OverNetwork,
	adminRChan chan<- Udp, sendBackupToAckChan <-chan BackUp, backupRChan chan<- BackUp, messageSenderChan chan<- OverNetwork, backupSenderChan chan<- BackUp) {
	ownID := IDInput
	var peers []int
	const timeout = 3000 * time.Millisecond

	numberOfNewMessages := 1 // ENDRE NAVN
	//numberOfTimeouts := 3 // Bør tas vekk når sequence number? eller settes til numberOfTimeouts igjen når en slettes? <- kanskje bedre

	var msgAcks []Ack
	var ownSeqStart int
	seqs := make([]int, MAX_N_LIFTS)

	msgAckTimer := time.NewTimer(timeout)

	for {
		select {
		case cP := <-ackCurrentPeersChan:
			peers = cP.Peers
			for i, ack := range msgAcks {
				var temp []int //Usikker på om temp bør hete noe annet.
				for _, peer := range peers {
					for _, acker := range ack.Ackers {
						if peer == acker {
							temp = append(temp, acker)
						}
					}
				}
				msgAcks[i].Ackers = temp
			}
			ownSeqStart = seqs[ownID]

			fmt.Println("NW::senAcks: Mottatt currentPeers, ny peers: ", peers)

		case msgToSend := <-adminToAckChan:
			//fmt.Println("NW::senAcks: Message to send out: ", msgToSend)
			var newAck Ack
			newAck.Message = msgToSend
			newAck.SequenceStart = ownSeqStart
			newAck.SequenceNumber = seqs[ownID]
			//newAck.Counter = numberOfTimeouts

			msgAcks = append(msgAcks, newAck)
			//fmt.Println("NW::senAcks: msgAcks etter msgToSend er lagt til: ", msgAcks)
			var newMessage OverNetwork
			newMessage.Message = msgToSend
			newMessage.SequenceStart = ownSeqStart
			newMessage.SequenceNumber = seqs[ownID]
			for i := 0; i < numberOfNewMessages; i++ {
				messageSenderChan <- newMessage
			}
			seqs[ownID]++


		case recvMsg := <-helloRx:
			//fmt.Println("NW::senAcks: Message received, MELLOMROM, current msgAcks: ", recvMsg, msgAcks)

			// Tenker at det her fungerer nå, men sent, så sikkert lurt å sjekke igjen når en er våken.
			if recvMsg.ThisIsAnAck {
				// Må være fra noen andre. Finn Message (Udp) i msgAcks, så sjekke om AckersID
				// er i Ackers[]. Hvis ikke, legg til. Så sjekke om len(msgAcks)>= len(peers) -1. If so,
				// send til admin, slett ack fra msgAcks.
				if recvMsg.AckersID != ownID {
					var indexOfMessagesToDelete []int
					if recvMsg.Message.ID == ownID { // Da er det ack på vår melding
						for i, ack := range msgAcks {
							if ack.Message == recvMsg.Message {
								alreadyAcked := false
								for _, acker := range ack.Ackers {
									if acker == recvMsg.AckersID { //Sikkert forbedringspotensiale med navn her
										alreadyAcked = true
									}
								}
								if !alreadyAcked {
									msgAcks[i].Ackers = append(msgAcks[i].Ackers, recvMsg.AckersID)
								}
								if len(peers) > 1 {
									if len(msgAcks[i].Ackers) >= len(peers)-1 {
										adminRChan <- ack.Message
										//msgAcks = append(msgAcks[:i], msgAcks[i+1:]...) Kan ikke stå her siden det endrer på for-løkka.
										indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
									} /*else if msgAcks[i].Counter <= 0 { // erstatt med or?
										adminRChan <- ack.Message
										//msgAcks = append(msgAcks[:i], msgAcks[i+1:]...) Kan ikke stå her siden det endrer på for-løkka.
										indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
									}*/
								}
							}
						}
					}
					//fmt.Println("NW: msgAcks in messages to delete: ", msgAcks)
					//fmt.Println("NW: indexOfMessagesToDelete: ", indexOfMessagesToDelete)
					for k, i := range indexOfMessagesToDelete {
						msgAcks = append(msgAcks[:i-k], msgAcks[i-k+1:]...)
					}

				}

			} else {
				// Sett ThisIsAnAck = true, AckersID = ownID. Acker på andre sine meldinger.
				if recvMsg.Message.ID != ownID {
					//fmt.Println("NW: RecvSeqStart, RecvSeqNr, seqs ", recvMsg.SequenceStart, recvMsg.SequenceNumber, seqs)
					if recvMsg.SequenceStart > seqs[recvMsg.Message.ID] {
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					}
					if recvMsg.SequenceNumber <= seqs[recvMsg.Message.ID] {
						//fmt.Println("NW: Fått melding, sender ack. Melding: ", recvMsg.Message)
						recvMsg.ThisIsAnAck = true
						recvMsg.AckersID = ownID
						messageSenderChan <- recvMsg
						if recvMsg.SequenceNumber == seqs[recvMsg.Message.ID] {
							adminRChan <- recvMsg.Message
							seqs[recvMsg.Message.ID]++
						}

					}

				}

			}


		case backup := <-sendBackupToAckChan:
			//fmt.Println("NW::senAcks: Backup to send: ", backup)

			// Trur virkelig den her bør endres, i hvert fall litt. Med mindre vi satse på at 99.9936 e bra nok
			// for det vi gjør.
			//backupToSend
			switch backup.SenderID {
			case ownID:
				//Send 2-5 times
				for i := 0; i < numberOfNewMessages; i++ { //Må sjekke om peer allerede er i aliveLifts
					backupSenderChan <- backup
				}

			//gotBackupFromSomeoneElse
			default:
				backupRChan <- backup
			}

		case <-msgAckTimer.C:
			//fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks at timeout: ", msgAcks)
			var indexOfMessagesToDelete []int
			if len(peers) > 1 {
				for i, ack := range msgAcks {
					//msgAcks[i].Counter--
					if len(ack.Ackers) >= len(peers)-1 {
						adminRChan <- ack.Message
						indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
					} //else {
					//msgAcks[i].Ackers = []int{} // ta bort?
					//}
				}

				/*else if msgAcks[i].Counter <= 0 {
					adminRChan <- ack.Message
					indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
					fmt.Println("NW::senAcks: Kaster ut en ack, denne: ", msgAcks[i])
				}*/

				for k, i := range indexOfMessagesToDelete {
					msgAcks = append(msgAcks[:i-k], msgAcks[i-k+1:]...)
				}

				for _, ack := range msgAcks { //Kanskje endre navn for kodekvalitet
					for i := 0; i < numberOfNewMessages; i++ {
						var newMessage OverNetwork
						newMessage.Message = ack.Message
						newMessage.SequenceStart = ack.SequenceStart
						newMessage.SequenceNumber = ack.SequenceNumber
						messageSenderChan <- newMessage
					}
				}
				msgAckTimer = time.NewTimer(timeout)
			} else { // Ingen andre funnet på nettet.

				for len(msgAcks) > 0 {
					adminRChan <- msgAcks[0].Message
					msgAcks = append(msgAcks[:0], msgAcks[1:]...)
					seqs[ownID]++
				}

				ownSeqStart = seqs[ownID]

			}
			//fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks AFTER LOOPS: ", msgAcks)

		}
	}

}

// Network starts the network module.
func Network(IDInput int, adminTChan <-chan Udp, adminRChan chan<- Udp, backupTChan <-chan BackUp,
	backupRChan chan<- BackUp, peerChangeChan chan<- Peer, peerInitializeChan chan<- CurrPeers) {

	ownID := IDInput

	init := true

	//var previousMessage []Udp // Eq to previousMessage := []Udp{} Brukes ikke akkurat nå
	// var previousBackup []BackUp LEGG INN HVIS DET FØLES LURT.

	var currentPeers []int //:= make([]int, 0, MAX_N_LIFTS)
	currentPeers = append(currentPeers, ownID)

	ackCurrentPeersChan := make(chan CurrPeers, 100) // ENDRE! LAG STRUCT FOR currentPeers.
	adminToAckChan := make(chan Udp, 100)
	sendBackupToAckChan := make(chan BackUp, 100)
	const timeout = 200 * time.Millisecond

	id := strconv.Itoa(ownID)

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)
	go peers.Transmitter(15640, id, peerTxEnable) //15647
	go peers.Receiver(15640, peerUpdateCh)        //15647


	helloTx := make(chan OverNetwork)
	helloRx := make(chan OverNetwork, 100)
	//go bcast.Transmitter(16570, id, helloTx) Dette var før, når Anders lagde funksjon. Funksjon fungerte ikke.
	//go bcast.Receiver(16570, id, false, helloRx) ---""----
	go bcast.Transmitter(16570, helloTx)
	go bcast.Receiver(16570, helloRx)

	backupTx := make(chan BackUp)
	backupRx := make(chan BackUp)
	go bcast.Transmitter(16571, backupTx)
	go bcast.Receiver(16571, backupRx)

	//lastMessage := Udp{NOT_VALID, "Test", NOT_VALID, NOT_VALID} // Eller bare tom?

	fmt.Println("NW: Started")

	messageSenderChan := make(chan OverNetwork, 100)
	backupSenderChan := make(chan BackUp, 100)
	go messageSender(helloTx, messageSenderChan)
	go backupSender(backupTx, backupSenderChan)

	go checksIncomingMessages(IDInput, ackCurrentPeersChan, adminToAckChan, helloRx,
		adminRChan, sendBackupToAckChan, backupRChan, messageSenderChan, backupSenderChan)
	/*
		c := CurrPeers{}
		c.Peers = currentPeers
		ackCurrentPeersChan <- c
		Skal gå bra uten det her: fikses lengre ned
	*/

	for {
		select {
		case backupToSend := <-backupTChan:
			fmt.Println("NW: backupToSend: ", backupToSend)
			sendBackupToAckChan <- backupToSend
		//Legg til case for tatt imot backup en plass

	case recievedBackup := <-backupRx:
			fmt.Println("NW: gotBackupFromSomeoneElse: ", recievedBackup)
			if recievedBackup.SenderID != ownID {
				sendBackupToAckChan <- recievedBackup
			} //Legg til previousBackup og test om den har blitt mottatt tidligere for bedre kode.

		case u := <-adminTChan:
			// NB!!! Send rett tilbake hvis ingen andre på nett.

			//if lastMessage != u {
				//fmt.Println("NW: len(currentPeers)", len(currentPeers))
				if len(currentPeers) == 1 {

					outsideDownPressed := (u.Type == "ButtonPressed") && (u.ExtraInfo == BUTTON_CALL_DOWN)
					outsideUpPressed := (u.Type == "ButtonPressed") && (u.ExtraInfo == BUTTON_CALL_UP)

					if !outsideDownPressed && !outsideUpPressed {
						adminRChan <- u
					}
					//fmt.Println("NW: (single): Melding", u)
				} else {

					adminToAckChan <- u
					//fmt.Println("NW: (not alone): Melding", u)

					//lastMessage = u
				}
			//}


		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			if init {
				var peersOnlineAtConnection CurrPeers
				var peersInt []int
				for i := range p.Peers {
					peerIDInt, _ := strconv.Atoi(p.Peers[i])
					peersInt = append(peersInt, peerIDInt)
				}
				peersOnlineAtConnection.Peers = peersInt
				peerInitializeChan <- peersOnlineAtConnection
				init = false
			}

			if len(p.New) > 0 {
				fmt.Println("NW: Mottatt New: ", p.New)
				fmt.Println("NW: currentPeers når mottatt: ", currentPeers)
				newID, _ := strconv.Atoi(p.New)
				if newID != ownID { // Må endres! vil tydeligvis ha egen id sent til admin i hvert fall.
					currentPeers = append(currentPeers, newID)
					sort.Slice(currentPeers, func(i, j int) bool { return currentPeers[i] < currentPeers[j] })
					peerChangeChan <- Peer{"New", newID}

					//Needs new name
					var currPeersToAck CurrPeers
					currPeersToAck.Peers = currentPeers
					ackCurrentPeersChan <- currPeersToAck
				}
				fmt.Println("NW: currentPeers etter mottatt: ", currentPeers)

			}

			if len(p.Lost) > 0 {
				var lostSlice []int
				fmt.Println("NW: Mottatt Lost: ", p.Lost)
				fmt.Println("NW: currentPeers når mottatt Lost: ", currentPeers)
				for i := range p.Lost {
					lostIDInt, _ := strconv.Atoi(p.Lost[i])
					lostSlice = append(lostSlice, lostIDInt)
				}

				for _, lostID := range lostSlice {
					for i, previouslyAliveID := range currentPeers {
						if lostID == previouslyAliveID { //Assumes you never get your own ID here, but haven't tested...
							currentPeers = append(currentPeers[:i], currentPeers[i+1:]...) // endre, bør tas bort på annen måte?
							peerChangeChan <- Peer{"Lost", lostID}
						}
					}

				}

				//Needs some new name
				var currPeersToAck CurrPeers
				currPeersToAck.Peers = currentPeers
				ackCurrentPeersChan <- currPeersToAck
				fmt.Println("NW: currentPeers etter lost er tatt ut: ", currentPeers)
			}


		}
	}
}

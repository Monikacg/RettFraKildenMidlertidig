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
		case backup := <-backupSenderChan:
			//fmt.Println("NW::backupSender: Sending this: ", bu)
			backupTx <- backup
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func checksIncomingMessages(IDInput int, ackCurrentPeersChan <-chan CurrPeers, adminToAckChan <-chan Udp, helloRx <-chan OverNetwork,
	adminRChan chan<- Udp, sendBackupToAckChan <-chan BackUp, backupRChan chan<- BackUp, messageSenderChan chan<- OverNetwork, backupSenderChan chan<- BackUp) {
	ownID := IDInput
	var peers []int
	const timeout = 3000 * time.Millisecond

	var msgAcks []Ack
	var ownSeqStart int
	seqs := make([]int, MAX_N_LIFTS)
	lastSeqsStartsRecv := make([]int, MAX_N_LIFTS) // TO REMOVE PROBLEM WHEN ONE LIFT GETS BACK ONLINE.

	resendTimer := time.NewTimer(timeout)

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
			//ownSeqStart = seqs[ownID] // NB! FEIL? Trengs ikke for oss. Kan bli feil hvis det her legges til.

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

			// Send message
			var newMessage OverNetwork
			newMessage.Message = msgToSend
			newMessage.SequenceStart = ownSeqStart
			newMessage.SequenceNumber = seqs[ownID]
			messageSenderChan <- newMessage
			seqs[ownID]++

		case recvMsg := <-helloRx:
			//fmt.Println("NW::senAcks: Message received, MELLOMROM, current msgAcks: ", recvMsg, msgAcks)

			if recvMsg.ThisIsAnAck {
				// Checks that it is from someone else and that it is originally sent out by this lift.
				if recvMsg.AckersID != ownID {
					var indicesOfMessagesToRemove []int
					if recvMsg.Message.ID == ownID {

						for i, ack := range msgAcks {
							if ack.Message == recvMsg.Message {
								alreadyAcked := false
								for _, acker := range ack.Ackers {
									if acker == recvMsg.AckersID {
										alreadyAcked = true
									}
								}
								if !alreadyAcked {
									msgAcks[i].Ackers = append(msgAcks[i].Ackers, recvMsg.AckersID)
								}
								if len(peers) > 1 {
									if len(msgAcks[i].Ackers) >= len(peers)-1 {
										adminRChan <- ack.Message
										indicesOfMessagesToRemove = append(indicesOfMessagesToRemove, i)
									} else {
										break // Assures that messages return in correct order
									} /*else if msgAcks[i].Counter <= 0 { // erstatt med or?
										adminRChan <- ack.Message
										//msgAcks = append(msgAcks[:i], msgAcks[i+1:]...) Kan ikke stå her siden det endrer på for-løkka.
										indicesOfMessagesToRemove = append(indicesOfMessagesToRemove, i)
									}*/
								}
							}
						}
					}
					//fmt.Println("NW: msgAcks in messages to delete: ", msgAcks)
					//fmt.Println("NW: indicesOfMessagesToRemove: ", indicesOfMessagesToRemove)
					for k, i := range indicesOfMessagesToRemove {
						msgAcks = append(msgAcks[:i-k], msgAcks[i-k+1:]...)
						ownSeqStart++
					}

				}

			} else {
				if recvMsg.Message.ID != ownID {
					// New message from someone else

					fmt.Println("NW: Got message from someone else. Sender ID::SequenceStart::SequenceNumber::seqs::lastSeqs at beginning", recvMsg.Message.ID, recvMsg.SequenceStart, recvMsg.SequenceNumber, seqs, lastSeqsStartsRecv)
					if recvMsg.SequenceStart > seqs[recvMsg.Message.ID] { // Skal ikke skje, bare for å være sikker.
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					} else if recvMsg.SequenceStart < lastSeqsStartsRecv[recvMsg.Message.ID] { // NOTE: IKKE TESTET
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					}

					lastSeqsStartsRecv[recvMsg.Message.ID] = recvMsg.SequenceStart // NEW

					fmt.Println("NW: After check: seqs::lastSeqs", seqs, lastSeqsStartsRecv)
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

			switch backup.SenderID {
			case ownID:
				//Send 2-5 times
				for i := 0; i < 1; i++ {
					backupSenderChan <- backup
				}
			default:
				backupRChan <- backup
			}

		case <-resendTimer.C:
			//fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks at timeout: ", msgAcks)
			var indicesOfMessagesToRemove []int
			if len(peers) > 1 {
				for i, ack := range msgAcks {
					//msgAcks[i].Counter--
					if len(ack.Ackers) >= len(peers)-1 {
						adminRChan <- ack.Message
						indicesOfMessagesToRemove = append(indicesOfMessagesToRemove, i)
					} else {
						break // Assures that messages return in correct order
					}
				}

				/*else if msgAcks[i].Counter <= 0 {
					adminRChan <- ack.Message
					indicesOfMessagesToRemove = append(indicesOfMessagesToRemove, i)
					fmt.Println("NW::senAcks: Kaster ut en ack, denne: ", msgAcks[i])
				}*/

				for k, i := range indicesOfMessagesToRemove {
					msgAcks = append(msgAcks[:i-k], msgAcks[i-k+1:]...)
					ownSeqStart++
				}

				// Resend all not acked.
				for _, ack := range msgAcks {
					var newMessage OverNetwork
					newMessage.Message = ack.Message
					newMessage.SequenceStart = ownSeqStart // ENDRET FRA FORRIGE GANG.
					newMessage.SequenceNumber = ack.SequenceNumber
					messageSenderChan <- newMessage
				}
				resendTimer = time.NewTimer(timeout)
			} else { // No one else online

				for len(msgAcks) > 0 {
					adminRChan <- msgAcks[0].Message
					msgAcks = append(msgAcks[:0], msgAcks[1:]...)
					ownSeqStart++ //Used to be seqs[ownID]++, tror ikke det var riktig
				}

				//ownSeqStart = seqs[ownID] trengs ikke det her

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

	var currentPeers []int
	currentPeers = append(currentPeers, ownID)

	ackCurrentPeersChan := make(chan CurrPeers, 100)
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

	//fmt.Println("NW: Started")

	messageSenderChan := make(chan OverNetwork, 100)
	backupSenderChan := make(chan BackUp, 100)
	go messageSender(helloTx, messageSenderChan)
	go backupSender(backupTx, backupSenderChan)

	go checksIncomingMessages(IDInput, ackCurrentPeersChan, adminToAckChan, helloRx,
		adminRChan, sendBackupToAckChan, backupRChan, messageSenderChan, backupSenderChan)

	for {
		select {
		case backupToSend := <-backupTChan:
			fmt.Println("NW: backupToSend: ", backupToSend)
			sendBackupToAckChan <- backupToSend

		case recievedBackup := <-backupRx:
			fmt.Println("NW: recievedBackup: ", recievedBackup)
			if recievedBackup.SenderID != ownID {
				sendBackupToAckChan <- recievedBackup
			}

		case newMessageToSend := <-adminTChan:
			if len(currentPeers) == 1 {

				outsideDownPressed := (newMessageToSend.Type == "ButtonPressed") && (newMessageToSend.ExtraInfo == BUTTON_CALL_DOWN)
				outsideUpPressed := (newMessageToSend.Type == "ButtonPressed") && (newMessageToSend.ExtraInfo == BUTTON_CALL_UP)

				if !outsideDownPressed && !outsideUpPressed {
					adminRChan <- newMessageToSend
				}
				//fmt.Println("NW: (single): Melding", u)
			} else {

				adminToAckChan <- newMessageToSend
				//fmt.Println("NW: (not alone): Melding", u)
			}

		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			// If det første i admin ikke trengs, trengs ikke denne løkka heller
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

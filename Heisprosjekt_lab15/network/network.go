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

func messageSender(messageTransmitChan chan<- Broadcast, messageSenderChan <-chan Broadcast) {
	for {
		select {
		case msg := <-messageSenderChan:
			//fmt.Println("NW::messageSender: Sending this: ", msg)
			messageTransmitChan <- msg
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func backupSender(backupTransmitChan chan<- BackUp, backupSenderChan <-chan BackUp) {
	for {
		select {
		case backup := <-backupSenderChan:
			//fmt.Println("NW::backupSender: Sending this: ", bu)
			backupTransmitChan <- backup
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func checksIncomingMessages(IDInput int, sendAliveLiftListToCheckIncomingMessagesChan <-chan Lifts, messagesToSendChan <-chan Message, messageReceiveChan <-chan Broadcast,
	incomingMessageChan chan<- Message, messageSenderChan chan<- Broadcast) {
	ownID := IDInput
	var aliveLifts []int
	const timeout = 3000 * time.Millisecond

	var msgAcks []Ack
	var ownSeqStart int
	seqs := make([]int, MAX_N_LIFTS)
	lastSeqsStartsRecv := make([]int, MAX_N_LIFTS) // TO REMOVE PROBLEM WHEN ONE LIFT GETS BACK ONLINE.

	resendTimer := time.NewTimer(timeout)

	for {
		select {
		case al := <-sendAliveLiftListToCheckIncomingMessagesChan:
			aliveLifts = al.AliveLifts
			for i, ack := range msgAcks {
				var aliveAckers []int
				for _, lift := range aliveLifts {
					for _, acker := range ack.Ackers {
						if lift == acker {
							aliveAckers = append(aliveAckers, acker)
						}
					}
				}
				msgAcks[i].Ackers = aliveAckers
			}

			fmt.Println("NW::senAcks: Mottatt aliveLifts, ny peers: ", aliveLifts)

		case msgToSend := <-messagesToSendChan:
			//fmt.Println("NW::senAcks: Message to send out: ", msgToSend)
			var newAck Ack
			newAck.Message = msgToSend
			newAck.SequenceNumber = seqs[ownID]

			msgAcks = append(msgAcks, newAck)
			//fmt.Println("NW::senAcks: msgAcks etter msgToSend er lagt til: ", msgAcks)

			// Send message
			var newMessage Broadcast
			newMessage.Message = msgToSend
			newMessage.SequenceStart = ownSeqStart
			newMessage.SequenceNumber = seqs[ownID]
			messageSenderChan <- newMessage
			seqs[ownID]++

		case recvMsg := <-messageReceiveChan:
			//fmt.Println("NW::senAcks: Message received, MELLOMROM, current msgAcks: ", recvMsg, msgAcks)

			if recvMsg.ThisIsAnAck {
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
						}
					}
					for len(msgAcks) > 0 {
						if len(msgAcks[0].Ackers) >= len(aliveLifts)-1 {
							incomingMessageChan <- msgAcks[0].Message
							msgAcks = append(msgAcks[:0], msgAcks[1:]...)
							ownSeqStart++
						} else {
							break // Assures that messages return in correct order
						}
					}

				}

			} else {
				if recvMsg.Message.ID != ownID {
					// New message from someone else

					fmt.Println("NW: Got message from someone else. Sender ID::SequenceStart::SequenceNumber::seqs::lastSeqs at beginning", recvMsg.Message.ID, recvMsg.SequenceStart, recvMsg.SequenceNumber, seqs, lastSeqsStartsRecv)
					if recvMsg.SequenceStart > seqs[recvMsg.Message.ID] {
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					} else if recvMsg.SequenceStart < lastSeqsStartsRecv[recvMsg.Message.ID] {
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					}

					lastSeqsStartsRecv[recvMsg.Message.ID] = recvMsg.SequenceStart

					fmt.Println("NW: After check: seqs::lastSeqs", seqs, lastSeqsStartsRecv)
					if recvMsg.SequenceNumber <= seqs[recvMsg.Message.ID] {
						//fmt.Println("NW: Fått melding, sender ack. Melding: ", recvMsg.Message)
						recvMsg.ThisIsAnAck = true
						recvMsg.AckersID = ownID
						messageSenderChan <- recvMsg
						if recvMsg.SequenceNumber == seqs[recvMsg.Message.ID] {
							incomingMessageChan <- recvMsg.Message
							seqs[recvMsg.Message.ID]++
						}

					}

				}

			}

		case <-resendTimer.C:
			//fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks at timeout: ", msgAcks)
			if len(aliveLifts) > 1 {
				for len(msgAcks) > 0 {
					if len(msgAcks[0].Ackers) >= len(aliveLifts)-1 {
						incomingMessageChan <- msgAcks[0].Message
						msgAcks = append(msgAcks[:0], msgAcks[1:]...)
						ownSeqStart++
					} else {
						break // Assures that messages return in correct order
					}
				}

				// Resend all not acked.
				for _, ack := range msgAcks {
					var newMessage Broadcast
					newMessage.Message = ack.Message
					newMessage.SequenceStart = ownSeqStart
					newMessage.SequenceNumber = ack.SequenceNumber
					messageSenderChan <- newMessage
				}
				resendTimer = time.NewTimer(timeout)
			} else { // No one else online

				for len(msgAcks) > 0 {
					incomingMessageChan <- msgAcks[0].Message
					msgAcks = append(msgAcks[:0], msgAcks[1:]...)
					ownSeqStart++
				}

			}
			//fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks AFTER LOOPS: ", msgAcks)

		}
	}

}


func Network(IDInput int, outgoingMessageChan <-chan Message, incomingMessageChan chan<- Message, outgoingBackupChan <-chan BackUp,
	incomingBackupChan chan<- BackUp, aliveLiftChangeChan chan<- ChangedLift) {

	ownID := IDInput

	var aliveLifts []int
	const timeout = 200 * time.Millisecond


	sendAliveLiftListToCheckIncomingMessagesChan := make(chan Lifts, 100)
	messagesToSendChan := make(chan Message, 100)



	id := strconv.Itoa(ownID)
	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)
	go peers.Transmitter(15640, id, peerTxEnable)
	go peers.Receiver(15640, peerUpdateCh)


	messageTransmitChan := make(chan Broadcast)
	messageReceiveChan := make(chan Broadcast, 100)
	go bcast.Transmitter(16570, messageTransmitChan)
	go bcast.Receiver(16570, messageReceiveChan)


	backupTransmitChan := make(chan BackUp)
	backupReceiveChan := make(chan BackUp)
	go bcast.Transmitter(16571, backupTransmitChan)
	go bcast.Receiver(16571, backupReceiveChan)


	messageSenderChan := make(chan Broadcast, 100)
	backupSenderChan := make(chan BackUp, 100)
	go messageSender(messageTransmitChan, messageSenderChan)
	go backupSender(backupTransmitChan, backupSenderChan)



	go checksIncomingMessages(IDInput, sendAliveLiftListToCheckIncomingMessagesChan, messagesToSendChan, messageReceiveChan,
		incomingMessageChan, messageSenderChan)


	for {
		select {
		case backupToSend := <-outgoingBackupChan:
			fmt.Println("NW: backupToSend: ", backupToSend)

			// This will give 99.968% chance of at least one message making it through with a package loss percentage of 20%.
			// Implementing acking the same way as we have with normal messages would be preferable (would make us closer to 100% certain that
			// a message get through), but we're a bit short on time and a 0.032% risk for a loss seemed reasonable.
			for i := 0; i < 5; i++ {
				backupSenderChan <- backupToSend
			}

		case receivedBackup := <-backupReceiveChan:
			fmt.Println("NW: receivedBackup: ", receivedBackup)
			if receivedBackup.SenderID != ownID {
				incomingBackupChan <- receivedBackup
			}

		case newMessageToSend := <-outgoingMessageChan:
			if len(aliveLifts) <= 1 {

				outsideDownPressed := (newMessageToSend.Info == "Button pressed") && (newMessageToSend.ButtonType == BUTTON_CALL_DOWN)
				outsideUpPressed := (newMessageToSend.Info == "Button pressed") && (newMessageToSend.ButtonType == BUTTON_CALL_UP)

				if !outsideDownPressed && !outsideUpPressed {
					incomingMessageChan <- newMessageToSend
				}
				//fmt.Println("NW: (single): Melding", u)
			} else {

				messagesToSendChan <- newMessageToSend
				//fmt.Println("NW: (not alone): Melding", u)
			}

		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)


			if len(p.New) > 0 {
				fmt.Println("NW: Mottatt New: ", p.New)
				fmt.Println("NW: aliveLifts når mottatt: ", aliveLifts)
				newID, _ := strconv.Atoi(p.New)
				aliveLifts = append(aliveLifts, newID)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })
				aliveLiftChangeChan <- ChangedLift{"New", newID}


				var currentlyAliveLifts Lifts
				currentlyAliveLifts.AliveLifts = aliveLifts
				sendAliveLiftListToCheckIncomingMessagesChan <- currentlyAliveLifts
				fmt.Println("NW: aliveLifts etter mottatt: ", aliveLifts)

			}

			if len(p.Lost) > 0 {
				var lostSlice []int
				fmt.Println("NW: Mottatt Lost: ", p.Lost)
				fmt.Println("NW: aliveLifts når mottatt Lost: ", aliveLifts)
				for i := range p.Lost {
					lostIDInt, _ := strconv.Atoi(p.Lost[i])
					lostSlice = append(lostSlice, lostIDInt)
				}

				lostDelCount := 0
				for i, lostID := range lostSlice {
					for j, previouslyAliveID := range aliveLifts {
						if lostID == previouslyAliveID {
							aliveLifts = append(aliveLifts[:j], aliveLifts[j+1:]...)
							aliveLiftChangeChan <- ChangedLift{"Lost", lostID}
							i--
							lostDelCount++
							break
						}
					}
					if len(aliveLifts) == i+lostDelCount-1 {
						break
					}

				}

				var currentlyAliveLifts Lifts
				currentlyAliveLifts.AliveLifts = aliveLifts
				sendAliveLiftListToCheckIncomingMessagesChan <- currentlyAliveLifts
				fmt.Println("NW: aliveLifts etter lost er tatt ut: ", aliveLifts)
			}

		}
	}
}

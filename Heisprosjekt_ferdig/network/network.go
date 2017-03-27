package network

import (
	"strconv"
	"time"
	"sort"

	. "../definitions"
	"./bcast"
	"./peers"
)

// All functions used by this file (found in the peers, localip, conn and bcast folders)
// were made by github.com/klasbo. See https://github.com/TTK4145/Network-go
// The functions in this file (network.go) are made by us.

func messageSender(messageTransmitChan chan<- Broadcast, messageSenderChan <-chan Broadcast) {
	for {
		select {
		case msg := <-messageSenderChan:
			messageTransmitChan <- msg
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func backupSender(backupTransmitChan chan<- BackUp, backupSenderChan <-chan BackUp) {
	for {
		select {
		case backup := <-backupSenderChan:
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
	lastSeqsStart := make([]int, MAX_N_LIFTS)

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

		case msgToSend := <-messagesToSendChan:
			var newAck Ack
			newAck.Message = msgToSend
			newAck.SequenceNumber = seqs[ownID]

			msgAcks = append(msgAcks, newAck)

			// Send message
			var newMessage Broadcast
			newMessage.Message = msgToSend
			newMessage.SequenceStart = ownSeqStart
			newMessage.SequenceNumber = seqs[ownID]
			messageSenderChan <- newMessage
			seqs[ownID]++

		case recvMsg := <-messageReceiveChan:

			if recvMsg.ThisIsAnAck {
				if recvMsg.Message.ID == ownID {
					// Someone (will always be someone else) sending an ack on a message we sent out.
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
					if recvMsg.SequenceStart > seqs[recvMsg.Message.ID] {
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					} else if recvMsg.SequenceStart < lastSeqsStart[recvMsg.Message.ID] {
						seqs[recvMsg.Message.ID] = recvMsg.SequenceStart
					}

					lastSeqsStart[recvMsg.Message.ID] = recvMsg.SequenceStart

					if recvMsg.SequenceNumber <= seqs[recvMsg.Message.ID] {
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

		}
	}

}



func Network(IDInput int, outgoingMessageChan <-chan Message, incomingMessageChan chan<- Message, outgoingBackupChan <-chan BackUp,
	incomingBackupChan chan<- BackUp, aliveLiftChangeChan chan<- ChangedLift) {

	ownID := IDInput

	var aliveLifts []int

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
			// This will give 99.968% chance of at least one message making it through with a package loss percentage of 20%.
			// Implementing acking the same way as we have with normal messages would be preferable (would make us closer to 100% certain that
			// a message get through), but we're a bit short on time and a 0.032% risk for a loss seemed reasonable.
			for i := 0; i < 5; i++ {
				backupSenderChan <- backupToSend
			}

		case receivedBackup := <-backupReceiveChan:
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
			} else {
				messagesToSendChan <- newMessageToSend
			}

		case p := <-peerUpdateCh:
			if len(p.New) > 0 {
				newID, _ := strconv.Atoi(p.New)
				aliveLifts = append(aliveLifts, newID)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })
				aliveLiftChangeChan <- ChangedLift{"New", newID}

				var currentlyAliveLifts Lifts
				currentlyAliveLifts.AliveLifts = aliveLifts
				sendAliveLiftListToCheckIncomingMessagesChan <- currentlyAliveLifts
			}

			if len(p.Lost) > 0 {
				var lostSlice []int
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
					if len(aliveLifts) == i+lostDelCount-1 { // Deletes indices in the slice we are iterating over, so have to break manually at the end.
						break
					}

				}

				var currentlyAliveLifts Lifts
				currentlyAliveLifts.AliveLifts = aliveLifts
				sendAliveLiftListToCheckIncomingMessagesChan <- currentlyAliveLifts
			}

		}
	}

}

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

// Test lastMessage der nede (i play.golang)

func messageSender(helloTx chan<- OverNetwork, messageSenderChan <-chan OverNetwork) {
	for {
		select {
		case msg := <-messageSenderChan:
			fmt.Println("NW::messageSender: Sending this: ", msg)
			helloTx <- msg
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func backupSender(backupTx chan<- BackUp, backupSenderChan <-chan BackUp) {
	for {
		select {
		case bu := <-backupSenderChan:
			fmt.Println("NW::backupSender: Sending this: ", bu)
			backupTx <- bu
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// VIKTIGST Å TEST OM MSGACKTIMER SLÅR UT RIKTIG NÅR DEN BLIR RESATT SÅ OFTE.
func sendAcks(IDInput int, ackCurrentPeersChan <-chan CurrPeers, adminToAckChan <-chan Udp, receivedFromOthersToAckChan <-chan OverNetwork,
	adminRChan chan<- Udp, sendBackupToAckChan <-chan BackUp, backupRChan chan<- BackUp, messageSenderChan chan<- OverNetwork, backupSenderChan chan<- BackUp) {
	ownID := IDInput
	var peers []int
	const timeout = 1000 * time.Millisecond

	numberOfNewMessages := 1 // ENDRE NAVN
	numberOfTimeouts := 2

	var lastBackupSent BackUp
	var msgAcks []Ack

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
			fmt.Println("NW::senAcks: Mottatt currentPeers, ny peers: ", peers)
		case msgToSend := <-adminToAckChan:
			fmt.Println("NW::senAcks: Message to send out: ", msgToSend)
			var newAck Ack
			newAck.Message = msgToSend
			newAck.Counter = numberOfTimeouts


			msgAcks = append(msgAcks, newAck)

			var newMessage OverNetwork
			newMessage.Message = msgToSend
			for i := 0; i < numberOfNewMessages; i++ {
				messageSenderChan <- newMessage
			}
			/* Siden de legger inn 15% packet loss (+et par prosent som kommer fra andre ting antar jeg):
			Send nok ganger til at sans for å komme frem er høy nok og ta det
			Sender en gang: 80% sjanse.
			Sender to ganger: 96% sjanse.
			Sender tre ganger: 99.2% sjanse.
			Sender fire ganger: 99.84% sjanse.
			Sender fem ganger: 99.968% sjanse.
			Sender seks ganger: 99.9936% sjanse.
			Antar her at ekstra unreliability fra UDP er 5% (total 20%). Skal være mer reliable enn dette. Alt over 4 virker som overkill.
			Rekkefølge kan være et problem for oss her.
			*/
			//time.Sleep(1 * time.Millisecond)
			//
			msgAckTimer = time.NewTimer(timeout)

		case recvMsg := <-receivedFromOthersToAckChan:
			fmt.Println("NW::senAcks: Message received: ", recvMsg)
			/*switch recvMsg.Message.ID {
			case ownID:
				if len(peers) > 1 {
					for i, ack := range msgAcks {
						if ownID == ack.Message.ID {
							ack.Counter++
							// Skal det under være med (det inni if)? If so, føles timer redundant
							//...som ville vært flott. Med mindre else-clausen blir utvidet.
							if ack.Counter >= len(peers)-1 {
								adminRChan <- ack.Message
								msgAcks = append(msgAcks[:i], msgAcks[i+1:]...)
							}
						}
					}
				}

				msgAckTimer = time.NewTimer(timeout) // Skeptisk til det her. Kan være at test nederst ville vært fint uten den delayen det her skaper
			default:
				messageSenderChan <- recvMsg
				// Sender bare 1 gang som det står nå. Dette er vår ack på en annen sin melding.
				// Hvis vi vil sende ack flere ganger for å garantere at den blir lagt inn, må vi ha
				// ID på ackingen.
			}*/

			//Hvis gammel versjon av bcast, legg inn test på AckersID hvis ack og ID på udp hvis ikke

			// Tenker at det her fungerer nå, men sent, så sikkert lurt å sjekke igjen når en er våken.
			if recvMsg.ThisIsAnAck {
				// Må være fra noen andre. Finn Message (Udp) i msgAcks, så sjekke om AckersID
				// er i Ackers[]. Hvis ikke, legg til. Så sjekke om len(msgAcks)>= len(peers) -1. If so,
				// send til admin, slett ack fra msgAcks.
				if recvMsg.AckersID != ownID {
					var indexOfMessagesToDelete []int
					if recvMsg.Message.ID == ownID { // Da er det ack på vår melding
						for i, ack := range msgAcks {
							msgAcks[i].Counter--
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
									} else if msgAcks[i].Counter <= 0 { // erstatt med or?
										adminRChan <- ack.Message
										//msgAcks = append(msgAcks[:i], msgAcks[i+1:]...) Kan ikke stå her siden det endrer på for-løkka.
										indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
									}
								}
							}
						}
					}
					for _, i := range indexOfMessagesToDelete {
						msgAcks = append(msgAcks[:i], msgAcks[i+1:]...)
					}

					//msgAckTimer = time.NewTimer(timeout)
				}

			} else {
				// ThisIsAnAck = True, AckersID = ownID. Acker på andre sine meldinger.
				if recvMsg.Message.ID != ownID {
					recvMsg.ThisIsAnAck = true
					recvMsg.AckersID = ownID
					adminRChan <- recvMsg.Message
					messageSenderChan <- recvMsg
				}

			}

			//Husk å plusse på 1 NÅR VÅR EGEN ID
			//Ellers send ut til de andre. Sjekk i Network sjekker sånn at vi ikke svarer på samme melding to ganger
			// (granted at de er etter hverandre -> svakhet i implementasjonen)

		case backup := <-sendBackupToAckChan:
			fmt.Println("NW::senAcks: Backup to send: ", backup)
			// Bare send samme antall ganger som vanlig ack og bli ferdig med det.
			// Står bare her i tilfelle vi vil legge inn acking på backup også. Akkurat nå skal
			// det bare sendes inn til admin (antar at alt kommer frem). Uten acking kan dette flyttes
			// til den andre loopen.

			// Trur virkelig den her bør endres, i hvert fall litt. Med mindre vi satse på at 99.9936 e bra nok
			// for det vi gjør.
			//backupToSend
			switch backup.SenderID {
			case ownID:
				//Send 2-5 times
				if backup != lastBackupSent {
					for i := 0; i < numberOfNewMessages; i++ {//Må sjekke om peer allerede er i aliveLifts
						backupSenderChan <- backup
					}
				}
				lastBackupSent = backup

			//gotBackupFromSomeoneElse
			default:
				backupRChan <- backup
			}

		case <-msgAckTimer.C:
			fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks at timeout: ", msgAcks)
			var indexOfMessagesToDelete []int
			if len(peers) > 1 {
				for i, ack := range msgAcks {
					msgAcks[i].Counter--
					if len(ack.Ackers) >= len(peers)-1 {
						adminRChan <- ack.Message
						indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
					} else if msgAcks[i].Counter <= 0 {
						adminRChan <- ack.Message
						indexOfMessagesToDelete = append(indexOfMessagesToDelete, i)
						fmt.Println("NW::senAcks: Kaster ut en ack, denne: ", msgAcks[i])
					} else {
						msgAcks[i].Ackers = []int{}
					}
				}
				for _, i := range indexOfMessagesToDelete {
					msgAcks = append(msgAcks[:i], msgAcks[i+1:]...)
				}

				for _, ack := range msgAcks { //Kanskje endre navn for kodekvalitet
					for i := 0; i < numberOfNewMessages; i++ {
						var newMessage OverNetwork
						newMessage.Message = ack.Message
						messageSenderChan <- newMessage
					}
				}
				msgAckTimer = time.NewTimer(timeout)
			} else {
				//fmt.Println("This means we have lost some other lift (or our connection)")
				//fmt.Println("Decide if we should remove all in acks (and previousMessages?) or not. ")
				//fmt.Println("PreviousMessages when Lost. acks here.")
				// Note: Hvis vi sender nok meldinger (gjør sannsynligheten for tap små)
				// ->Kan ta bort len(peers) > 1, da vil alle bli tatt bort når alene
				// Alternativet nå: Sletter alle i slicen. Må sende vår egen heis våre egne kommandoer
				for len(msgAcks) > 0 { //Kanskje endre navn for kodekvalitet
					//if ack.Message.ID == ownID { // Unødvendig, alle skal jo være våre meldinger
					adminRChan <- msgAcks[0].Message
					msgAcks = append(msgAcks[:0], msgAcks[1:]...) // Denne må løses med while-løkke. Her slettes alle.
					//}
					//msgAcks = append(msgAcks[:i], msgAcks[i+1:]...)
				}

			}
			fmt.Println("NW::senAcks: Got timeout (msgacks). MsgAcks AFTER LOOPS: ", msgAcks)

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
	// for test:
	//currentPeers = append(currentPeers, 1)

	ackCurrentPeersChan := make(chan CurrPeers, 100) // ENDRE! LAG STRUCT FOR currentPeers.
	adminToAckChan := make(chan Udp, 100)
	receivedFromOthersToAckChan := make(chan OverNetwork, 100)
	sendBackupToAckChan := make(chan BackUp, 100)
	//outPutCh := make(chan Udp, 100)
	const timeout = 200 * time.Millisecond

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)
	go peers.Transmitter(15640, id, peerTxEnable) //15647
	go peers.Receiver(15640, peerUpdateCh)        //15647

	/*
		Lag en struct som er Udp-strukten+ack-felt som sier hvem som vet om beskjeden, ok ide?
	*/

	helloTx := make(chan OverNetwork)
	helloRx := make(chan OverNetwork)
	//go bcast.Transmitter(16570, id, helloTx) Dette var før, når Anders lagde funksjon. Funksjon fungerte ikke.
	//go bcast.Receiver(16570, id, false, helloRx) ---""----
	go bcast.Transmitter(16570, helloTx)
	go bcast.Receiver(16570, helloRx)


	backupTx := make(chan BackUp)
	backupRx := make(chan BackUp)
	go bcast.Transmitter(16571, backupTx)
	go bcast.Receiver(16571, backupRx)

	//localAckCh := make(chan Udp, 100)

	//outPutCh := make(chan Udp, 100)

	lastMessage := Udp{NOT_VALID, "Test", NOT_VALID, NOT_VALID} // Eller bare tom?

	fmt.Println("NW: Started")

	messageSenderChan := make(chan OverNetwork, 100)
	backupSenderChan := make(chan BackUp, 100)
	go messageSender(helloTx, messageSenderChan)
	go backupSender(backupTx, backupSenderChan)

	go sendAcks(IDInput, ackCurrentPeersChan, adminToAckChan, receivedFromOthersToAckChan,
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

		case gotBackupFromSomeoneElse := <-backupRx:
			fmt.Println("NW: gotBackupFromSomeoneElse: ", gotBackupFromSomeoneElse)
			sendBackupToAckChan <- gotBackupFromSomeoneElse //Legg til previousBackup og test om den har blitt mottatt tidligere for bedre kode.

		case u := <-adminTChan:
			// Når en får beskjed fra admin om noe: legg inn i sende-struct,
			// create den siste lista, legg til sin egen ID som en som vet om,
			// send via bcast Transmitter. (helloTx-kanalen so far)

			// NB!!! Send rett tilbake hvis ingen andre på nett.

			if lastMessage != u {
				fmt.Println("NW: len(currentPeers)", len(currentPeers))
				if len(currentPeers) == 1 {
					//TA BORT YTRE KNAPPER
					//FUNGERER IKKE:  TAR IKKE BORT YTRE KNAPPER; TAR IKKE BORT NOEN.
					// De Morgan's law (bør fungere):

					outsideDownPressed := (u.Type == "ButtonPressed") && (u.ExtraInfo == BUTTON_CALL_DOWN)
					outsideUpPressed := (u.Type == "ButtonPressed") && (u.ExtraInfo == BUTTON_CALL_UP)

					if !outsideDownPressed && !outsideUpPressed {
						adminRChan <- u
					} else {
						//Ta bort før presentasjon
						fmt.Println("NW: I single elevator mode, pressing outer buttons doesn't do anything")
					}
					fmt.Println("NW: (single): Melding", u)
				} else {

					adminToAckChan <- u
					fmt.Println("NW: (not alone): Melding", u)

					lastMessage = u
				}
			}

		case recv := <-helloRx:
			fmt.Println("NW: Received from helloRx: ", recv)
			/*
			if recv.ThisIsAnAck {
				receivedFromOthersToAckChan <- recv
			} else if recv.Message.ID != ownID {
				receivedFromOthersToAckChan <- recv
			} */
			receivedFromOthersToAckChan <- recv

			//If vår ID, send to ack (receivedFromOthersToAckChan)
			/*switch recv.Message.ID { // SISTE KOMMENTAR: Tror ikke det her trengs, da alle vil sendes til ack.
			case ownID:
				receivedFromOthersToAckChan <- recv
			default:
				previouslyReceivedFrom := false
				for i, m := range previousMessage { // Går greit å iterere over tom slice også.
					if recv.ID == m.ID {
						previouslyReceivedFrom = true
						if recv != m {
							previousMessage = append(previousMessage[:i], previousMessage[i+1:]...)
							previousMessage = append(previousMessage, Udp{NOT_VALID, "NOT_VALID", NOT_VALID, NOT_VALID}) //What Udp message this is doesn't matter as it will not be added.
							copy(previousMessage[i+1:], previousMessage[i:])
							previousMessage[i] = recv
							receivedFromOthersToAckChan <- recv
							break // THIS WORKS
							// Skal dette bare sendes ut til nettet? Løsn: Nei, ser bra ut sånn
							// Men: Bør kanskje ha et Ack-felt som det står i OverNetwork så
							// vi vet om meldingen kommer fra kilden eller ikke => hjelper nok
							// når det gjelder rekkefølge. Legger da bare til de som IKKE har
							// Ack her (altså, Ack == false). Ack == true vil sendes til ackdel...
							// Nei. Ack == true && ID lik ownID => til ack => skal telle opp på Counter.
							// Ack == false => melding fra noen andre, håndteres som står her.
							// ^Legg til etter resten er fikset. Note2: Ack vil bare være i caset over => ubrukelig
							// MOST IMPORTANT NOTE: Trenger virkelig ack for å skille mellom egne meldinger og meldinger som sendes som retur.
							// Ack kan godt ha ID med seg også, slik at vi IKKE trenger dette med previousMessage/rekkefølge på meldingene
							// vil bli tatt hånd om.
						}
					}
				}
				if !previouslyReceivedFrom {
					previousMessage = append(previousMessage, recv)
					receivedFromOthersToAckChan <- recv
					// Skal dette bare sendes ut til nettet?
					// Ikke tidligere tatt fra -> Til admin. Men skal til ack først?
				}
			}
			fmt.Println("hei", recv)
			// Else, check om den er lik den siste meldingen vi fikk fra den heisen. If so,
			// gjør ingenting. If not, send ut + send til admin.
			*/

		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			if init {
				var peersOnlineAtConnection CurrPeers
				var peersInt []int
				for i := range p.Peers { // Need to make strings ints
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
							currentPeers = append(currentPeers[:i], currentPeers[i+1:]...)
							peerChangeChan <- Peer{"Lost", lostID}
						}
					}
					/*
						for i, m := range previousMessage {
							if lostID == m.ID {
								previousMessage = append(previousMessage[:i], previousMessage[i+1:]...)
								break
							}
						}
					*/
				}

				//Needs some new name
				var currPeersToAck CurrPeers
				// Alternativt: currPeersToAck := CurrPeers{} Gjør det samme
				currPeersToAck.Peers = currentPeers
				ackCurrentPeersChan <- currPeersToAck
				fmt.Println("NW: currentPeers etter lost er tatt ut: ", currentPeers)
			}

			//fmt.Printf("Received: %#v\n", a)
			// Når WhoKnowsAboutThis bare er -1, skal resten av structen sendes til admin
			// Ellers skal vi legge til vår egen ID i WhoKnowsAboutThis (hvis den ikke er der),
			// så sende meldingen (ellers) uendret ut igjen.

			// Husk å send på nytt
		}
		//time.Sleep(1 * time.Second)
	}
}

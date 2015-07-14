package main

import (
	"encoding/binary"
	"fmt"
	"github.com/HearthSim/hs-proto/go"
	"github.com/golang/protobuf/proto"
	"hash/fnv"
	"net"
	"io"
    "time"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = 1119
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type Service struct {
	Name string
    Id   uint32
}

func (s Service) GetHashedName() uint32 {
	h := fnv.New32a()
	h.Write([]byte(s.Name))
	return h.Sum32()
}


type ConnectionService struct{ Service }
type AuthServerService struct{ Service }
type AuthClientService struct{ Service }

type session struct {
	conn net.Conn
}

func HashToName (hash uint32) string {
    hashmap := map[uint32]string{
        3338259653: "bnet.protocol.game_master.GameFactorySubscriber",
        3213656212: "bnet.protocol.channel.ChannelSubscriber",
        2749215165: "bnet.protocol.friends.FriendsService",
        1864735251: "bnet.protocol.friends.FriendsNotify",
        3686756121: "bnet.protocol.challenge.ChallengeService",
        101490829: "bnet.protocol.channel.ChannelOwner",
        1898188341: "bnet.protocol.authentication.AuthenticationClient",
        1698982289: "bnet.protocol.connection.ConnectionService",
        3073563442: "bnet.protocol.channel.Channel",
        233634817: "bnet.protocol.authentication.AuthenticationServer",
        1423956503: "bnet.protocol.account.AccountNotify",
        1467132723: "bnet.protocol.game_master.GameMasterSubscriber",
        2165092757: "bnet.protocol.game_master.GameMaster",
        3151632159: "bnet.protocol.challenge.ChallengeNotify",
        1069623117: "bnet.protocol.game_utilities.GameUtilities",
        2198078984: "bnet.protocol.channel_invitation.ChannelInvitationService",
        4194801407: "bnet.protocol.presence.PresenceService",
        3788189352: "bnet.protocol.notification.NotificationListener",
        1658456209: "bnet.protocol.account.AccountService",
        3971904954: "bnet.protocol.resources.Resources",
        213793859: "bnet.protocol.notification.NotificationService",
    }
    if hashmap[hash] != "" {
        return hashmap[hash]
    }
    return "Unknown hash"
}

func (session *session) handleRequest(packet []byte) int {
	headerSize := binary.BigEndian.Uint16(packet)
	headerData := packet[2 : 2+int(headerSize)]
    fmt.Printf("%x\n", headerData)

	header := &hsproto.BnetProtocol_Header{}
	err := proto.Unmarshal(headerData, header)
	check(err)
    if header.GetStatus() != 0 {
        fmt.Println("header status != 0! :(");
    }

	packetEnd := 2 + int(headerSize) + int(header.GetSize())
	bodyData := packet[2+headerSize : packetEnd]

	if header.GetServiceId() == 0 && header.GetMethodId() == 1 {
		body := &hsproto.BnetProtocolConnection_ConnectRequest{}
		err = proto.Unmarshal(bodyData, body)
		check(err)

		// register services
		connService := ConnectionService{Service{"bnet.protocol.connection.ConnectionService", 0}}
		authServerService := AuthServerService{Service{"bnet.protocol.authentication.AuthenticationServer", 1}}
		authClientService := AuthClientService{Service{"bnet.protocol.authentication.AuthenticationClient", 255}}
		fmt.Printf("connService=%d, authServerService=%d, authClientService=%d\n",
			connService.GetHashedName(),
			authServerService.GetHashedName(),
			authClientService.GetHashedName(),
		)

		bindRequest := body.GetBindRequest()
		// iterate
		for _, importedHash := range bindRequest.GetImportedServiceHash() {
            fmt.Printf("Client imports service %d probably: %s\n", importedHash, HashToName(importedHash))
		}

		for _, export := range bindRequest.GetExportedService() {
            fmt.Printf("Client exports service id=%d, hash=%d probably: %s\n", export.GetId(), export.GetHash(), HashToName(export.GetHash()))
		}

		timestamp := uint64(time.Now().UnixNano() / 1000)
		epoch := uint32(time.Now().Unix())

		resp := &hsproto.BnetProtocolConnection_ConnectResponse{
			ServerId: &hsproto.BnetProtocol_ProcessId{
				Label: proto.Uint32(3868510373),
				Epoch: proto.Uint32(epoch),
			},
			ClientId: &hsproto.BnetProtocol_ProcessId{
				Label: proto.Uint32(1255760),
				Epoch: proto.Uint32(epoch),
			},
			BindResult: proto.Uint32(0),
			BindResponse: &hsproto.BnetProtocolConnection_BindResponse{
				ImportedServiceId: []uint32{
                    1,
                    2,
                    3,
                    4,
                    5,
                    6,
                    7,
                    8,
                    9,
                    10,
                    11,
                    12,
                },
			},
			ServerTime: proto.Uint64(timestamp),
		}

		data, err := proto.Marshal(resp)
		check(err)
		header := &hsproto.BnetProtocol_Header{
			ServiceId: proto.Uint32(254),
			MethodId:  proto.Uint32(1),
			Token:     proto.Uint32(header.GetToken()),
			Size:      proto.Uint32(uint32(len(data))),
		}

		session.writePacket(header, data)
	} else if header.GetServiceId() == 1 && header.GetMethodId() == 1 {
        fmt.Println("Auth Logon");
		body := &hsproto.BnetProtocolAuthentication_LogonRequest{}
		err = proto.Unmarshal(bodyData, body)
		check(err)

        header := &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(0),
            Status: proto.Uint32(0),
        }
        session.writePacket(header, make([]byte,0))

        resp := &hsproto.BnetProtocolAuthentication_LogonQueueUpdateRequest{
            Position: proto.Uint32(0),
            EstimatedTime: proto.Uint64(0),
            EtaDeviationInSec: proto.Uint64(0),
        }
        data, err := proto.Marshal(resp)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(1),
            MethodId:  proto.Uint32(12),
            Token:     proto.Uint32(0),
            Size:      proto.Uint32(uint32(len(data))),
        }
        session.writePacket(header, data);

        respUpd := &hsproto.BnetProtocolAuthentication_LogonUpdateRequest{
            ErrorCode: proto.Uint32(0),
        }
        data, err = proto.Marshal(respUpd)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(1),
            MethodId:  proto.Uint32(10),
            Token:     proto.Uint32(2),
            Size:      proto.Uint32(uint32(len(data))),
        }
        session.writePacket(header, data);

        respResult := &hsproto.BnetProtocolAuthentication_LogonResult {
            ErrorCode: proto.Uint32(0),
            Account: &hsproto.BnetProtocol_EntityId {
                High:  proto.Uint64(1),
                Low:   proto.Uint64(0),
            },
            GameAccount: []*hsproto.BnetProtocol_EntityId {
                &hsproto.BnetProtocol_EntityId {
                    High:  proto.Uint64(2),
                    Low:   proto.Uint64(0),
                },
            },
            ConnectedRegion: proto.Uint32(0),
        }
        data, err = proto.Marshal(respResult)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(1),
            MethodId:  proto.Uint32(5),
            Token:     proto.Uint32(3),
            Size:      proto.Uint32(uint32(len(data))),
        }
        session.writePacket(header, data);
    } else if header.GetServiceId() == 1 && header.GetMethodId() == 4 {
        fmt.Println("Auth SelectGameAccount");
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(0),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, make([]byte, 0));
    } else if header.GetServiceId() == 5 && header.GetMethodId() == 1 {
        fmt.Println("Presence Subscribe");
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(0),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, make([]byte, 0));
    } else if header.GetServiceId() == 11 && header.GetMethodId() == 30 {
        fmt.Println("Account GetAccountState");
        count := "EU"
        resp := &hsproto.BnetProtocolAccount_GetAccountStateResponse {
            State: &hsproto.BnetProtocolAccount_AccountState{
                AccountLevelInfo: &hsproto.BnetProtocolAccount_AccountLevelInfo {
                    Licenses: []*hsproto.BnetProtocolAccount_AccountLicense{
                        &hsproto.BnetProtocolAccount_AccountLicense {
                            Id: proto.Uint32(0),
                        },
                    },
                    DefaultCurrency: proto.Uint32(0),
                    Country: &count,
                    PreferredRegion: proto.Uint32(0),
                },
            },
        }
        data, err := proto.Marshal(resp)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(uint32(len(data))),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, data);
    } else if header.GetServiceId() == 9 && header.GetMethodId() == 1 {
        fmt.Println("Friends SubscribeToFriends");
        resp := &hsproto.BnetProtocolFriends_SubscribeToFriendsResponse {
            MaxFriends: proto.Uint32(100),
            MaxReceivedInvitations: proto.Uint32(42),
            MaxSentInvitations: proto.Uint32(7),
        }
        data, err := proto.Marshal(resp)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(uint32(len(data))),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, data);
    } else if header.GetServiceId() == 12 && header.GetMethodId() == 1 {
        fmt.Println("Resources GetContentHandle");
        contentRequest := &hsproto.BnetProtocolResources_ContentHandleRequest{}
        err := proto.Unmarshal(bodyData, contentRequest)
        fmt.Printf("%d %d\n",contentRequest.GetProgramId(), contentRequest.GetStreamId())
        resp := &hsproto.BnetProtocol_ContentHandle {
            Region: proto.Uint32(0),
            Usage: proto.Uint32(0),
            Hash: make([]byte, 0),
        }
        data, err := proto.Marshal(resp)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(uint32(len(data))),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, data);
    } else if header.GetServiceId() == 8 && header.GetMethodId() == 1 {
        fmt.Println("ChannelInvitation Subscribe");
        resp := &hsproto.BnetProtocolChannelInvitation_SubscribeResponse {
        }
        data, err := proto.Marshal(resp)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(uint32(len(data))),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, data);
    } else if header.GetServiceId() == 2 && header.GetMethodId() == 1 {
        fmt.Println("GameUtilities ClientRequest");
        clientRequest := &hsproto.BnetProtocolGameUtilities_ClientRequest{}
        proto.Unmarshal(bodyData, clientRequest)
        if len(clientRequest.GetAttribute()) != 2 {
            fmt.Println("Too many attributes in pegasus client request?");
        }
        packetType := int64(-1)
        for _, att := range clientRequest.GetAttribute() {
            fmt.Println(att.GetName())
            if att.GetName() == "p" {
                fmt.Printf("%x\n", att.GetValue().GetBlobValue())
                blob := att.GetValue().GetBlobValue()
                if len(blob) < 2 {
                    fmt.Println("blob is too short")
                } else {
                    packetType = int64(blob[0]) + int64(blob[1]) << 8
                }
            }
        }
        fmt.Printf("packet type %d\n", packetType)
        name := "it shouldn't matter"
        responseType := int64(packetType + 1) // looks like response type is request + 1
        resp := &hsproto.BnetProtocolGameUtilities_ClientResponse {
            Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                &hsproto.BnetProtocolAttribute_Attribute {
                    Name: &name,
                    Value: &hsproto.BnetProtocolAttribute_Variant {
                        //IntValue: &packetType,
                        IntValue: &responseType,
                    },
                },
                &hsproto.BnetProtocolAttribute_Attribute {
                    Name: &name,
                    Value: &hsproto.BnetProtocolAttribute_Variant {
                        BlobValue: []byte{},
                    },
                },
            },
        }
        data, err := proto.Marshal(resp)
        check(err)
        header = &hsproto.BnetProtocol_Header {
            ServiceId: proto.Uint32(254),
            Token:     proto.Uint32(header.GetToken()),
            Size:      proto.Uint32(uint32(len(data))),
            Status:    proto.Uint32(0),
        }
        session.writePacket(header, data);
    } else if header.GetServiceId() == 0 && header.GetMethodId() == 5 {
        fmt.Println("Keep alive!")
    } else {
        fmt.Printf("unsupported: %d, %d, bodylen: %d\n %x\nheader: %x\n", header.GetServiceId(), header.GetMethodId(), len(bodyData), string(bodyData[:]), string(headerData[:]))
	}
	return packetEnd
}

func createSession(conn net.Conn) session {
	s := session{conn}

	return s
}

func (session *session) writePacket(head *hsproto.BnetProtocol_Header, body []byte) {
	headerData, err := proto.Marshal(head)
	check(err)
	outPacket := make([]byte, 2+len(headerData)+len(body))
	binary.BigEndian.PutUint16(outPacket, uint16(len(headerData)))
	copy(outPacket[2:], headerData)
	copy(outPacket[2+len(headerData):], body)
	written, err := session.conn.Write(outPacket)
	if written != len(outPacket) {
		fmt.Println("didn't write full packet, fixme")
	}
	check(err)
}

func (session *session) serve() {
	defer session.conn.Close()

	buf := make([]byte, 4096)
	idx := 0
	for {
		read, err := session.conn.Read(buf[idx:])
        if err == io.EOF {
            fmt.Println("EOF?")
            break
        }
		check(err)
		totalProcessed := 0
		for read > totalProcessed {
			idx += read

			processed := session.handleRequest(buf[:idx])
			totalProcessed += processed
			idx -= processed
			if processed > 0 && totalProcessed < read {
				copy(buf[:len(buf)-processed], buf[processed:])
			} else {
				break
			}
		}
		if idx == len(buf) {
			buf = append(buf, make([]byte, len(buf)>>1)...)
		}
	}
}

func main() {
	// Listen for incoming connections
	hostname := fmt.Sprintf("%s:%d", CONN_HOST, CONN_PORT)
	tcpAddr, err := net.ResolveTCPAddr("tcp", hostname)
	check(err)
	sock, err := net.ListenTCP("tcp", tcpAddr)
	defer sock.Close()
	check(err)

	fmt.Printf("Listening on %s:%d ...\n", CONN_HOST, CONN_PORT)
	for {
		conn, err := sock.Accept()
		check(err)

		s := createSession(conn)
		go s.serve()
	}
}

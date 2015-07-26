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
    "runtime/debug"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = 1119
)

func check(e error) {
	if e != nil {
        debug.PrintStack()
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
    if len(packet) < 2 {
        return -1
    }
	headerSize := binary.BigEndian.Uint16(packet)
    if 2 + int(headerSize) > len(packet) {
        fmt.Printf("CO DO KURWY %d\n", int(headerSize))
        return -2
    }
	headerData := packet[2 : 2+int(headerSize)]
    fmt.Printf("%x\n", headerData)

	header := &hsproto.BnetProtocol_Header{}
	err := proto.Unmarshal(headerData, header)
	check(err)
    if header.GetStatus() != 0 {
        fmt.Println("header status != 0! :(");
    }

	packetEnd := 2 + int(headerSize) + int(header.GetSize())
    if packetEnd > len(packet) {
        fmt.Printf("SZTO? %d %d\n", header.GetSize(), packetEnd)
        return -3
    }
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
    } else if header.GetServiceId() == 5 && header.GetMethodId() == 3 {
        fmt.Println("Presence Update");
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
    } else if header.GetServiceId() == 11 && header.GetMethodId() == 34 {
        fmt.Println("Account GetGameSessionInfo");
        resp := &hsproto.BnetProtocolAccount_GetGameSessionInfoResponse {
            SessionInfo: &hsproto.BnetProtocolAccount_GameSessionInfo {
                StartTime: proto.Uint32(uint32(time.Now().Unix())),
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
        var pegasusData []byte;
        for _, att := range clientRequest.GetAttribute() {
            //fmt.Println(att.GetName())
            if att.GetName() == "p" {
                //fmt.Printf("%x\n", att.GetValue().GetBlobValue())
                blob := att.GetValue().GetBlobValue()
                if len(blob) < 2 {
                    fmt.Println("blob is too short")
                } else {
                    packetType = int64(blob[0]) + int64(blob[1]) << 8
                    pegasusData = blob[2:]
                }
            }
        }
        name := "it shouldn't matter"
        resp := &hsproto.BnetProtocolGameUtilities_ClientResponse {}
        if packetType == 314 {
            fmt.Println("pegasus 314 - subscribe")
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(315),
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
        } else if packetType == 303 {
            fmt.Println("pegasus 303 - get assets")
            assetResp := &hsproto.PegasusUtil_AssetsVersionResponse {
                Version: proto.Int32(7553),
            }
            assData, err := proto.Marshal(assetResp)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(304),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: assData,
                        },
                    },
                },
            }
        } else if packetType == 267 {
            fmt.Println("pegasus 267 - check account licenses")
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(325),
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
        } else if packetType == 276 {
            fmt.Println("pegasus 276 - check game licenses")
            licResp := &hsproto.PegasusUtil_CheckLicensesResponse {
                AccountLevel: proto.Bool(true),
                Success: proto.Bool(true),
            }
            licData, err := proto.Marshal(licResp)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(277),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: licData,
                        },
                    },
                },
            }
        } else if packetType == 205 {
            fmt.Println("pegasus 205 - update login")
            updResp := &hsproto.PegasusUtil_UpdateLoginComplete {
            }
            updData, err := proto.Marshal(updResp)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(307),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: updData,
                        },
                    },
                },
            }
        } else if packetType == 201 {
            fmt.Println("pegasus 201 - get account info")
            accountInfoReq := &hsproto.PegasusUtil_GetAccountInfo{}
            err := proto.Unmarshal(pegasusData, accountInfoReq)
            check(err)
            requestType := accountInfoReq.GetRequest()
            fmt.Println(requestType)
            attBlob := []byte{}
            respType := int64(505)
            if requestType == hsproto.PegasusUtil_GetAccountInfo_CAMPAIGN_INFO {
                profileProgress := &hsproto.PegasusUtil_ProfileProgress {
                    Progress: proto.Int64(6),
                    BestForge: proto.Int32(10),
                    LastForge: &hsproto.PegasusShared_Date {
                        Year: proto.Int32(2015),
                        Month: proto.Int32(3),
                        Day: proto.Int32(31),
                        Hours: proto.Int32(17),
                        Min: proto.Int32(3),
                        Sec: proto.Int32(54),
                    },
                }
                attBlob, err = proto.Marshal(profileProgress)
                check(err)
                respType = 233
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_BOOSTERS {
                boosters := &hsproto.PegasusUtil_BoosterList{}
                attBlob, err = proto.Marshal(boosters)
                check(err)
                respType = 224
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_FEATURES {
                features := &hsproto.PegasusUtil_GuardianVars {
                    ShowUserUI: proto.Int32(1),
                }
                attBlob, err = proto.Marshal(features)
                check(err)
                respType = 264
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_MEDAL_INFO {
                medalInfo := &hsproto.PegasusUtil_MedalInfo {
                    SeasonWins: proto.Int32(0),
                    Stars: proto.Int32(20),
                    Streak: proto.Int32(0),
                    StarLevel: proto.Int32(9),
                    LevelStart: proto.Int32(20),
                    LevelEnd: proto.Int32(3),
                    CanLose: proto.Bool(true),
                }
                attBlob, err = proto.Marshal(medalInfo)
                check(err)
                respType = 232
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_NOTICES {
                notices := &hsproto.PegasusUtil_ProfileNotices {}
                attBlob, err = proto.Marshal(notices)
                check(err)
                respType = 212
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_DECK_LIST {
                deckList := &hsproto.PegasusUtil_DeckList {}
                attBlob, err = proto.Marshal(deckList)
                check(err)
                respType = 202
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_COLLECTION {
                collection := &hsproto.PegasusUtil_Collection {}
                attBlob, err = proto.Marshal(collection)
                check(err)
                respType = 207
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_DECK_LIMIT {
                deckLimit := &hsproto.PegasusUtil_ProfileDeckLimit {
                    DeckLimit: proto.Int32(9),
                }
                attBlob, err = proto.Marshal(deckLimit)
                check(err)
                respType = 231
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_CARD_VALUES {
                cardValues := &hsproto.PegasusUtil_CardValues {
                    CardNerfIndex: proto.Int32(5),
                }
                attBlob, err = proto.Marshal(cardValues)
                check(err)
                respType = 260
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_ARCANE_DUST_BALANCE {
                arcaneDust := &hsproto.PegasusUtil_ArcaneDustBalance {
                    Balance: proto.Int64(1337),
                }
                attBlob, err = proto.Marshal(arcaneDust)
                check(err)
                respType = 262
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_NOT_SO_MASSIVE_LOGIN {
                notMassive := &hsproto.PegasusUtil_NotSoMassiveLoginReply {}
                attBlob, err = proto.Marshal(notMassive)
                check(err)
                respType = int64(hsproto.PegasusUtil_NotSoMassiveLoginReply_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_REWARD_PROGRESS {
                rewardProgress := &hsproto.PegasusUtil_RewardProgress {
                    SeasonEnd: &hsproto.PegasusShared_Date {
                        Year: proto.Int32(2015),
                        Month: proto.Int32(4),
                        Day: proto.Int32(30),
                        Hours: proto.Int32(22),
                        Min: proto.Int32(0),
                        Sec: proto.Int32(0),
                    },
                    WinsPerGold: proto.Int32(3),
                    GoldPerReward: proto.Int32(10),
                    MaxGoldPerDay: proto.Int32(100),
                    SeasonNumber: proto.Int32(18),
                    XpSoloLimit: proto.Int32(60),
                    MaxHeroLevel: proto.Int32(60),
                    NextQuestCancel: &hsproto.PegasusShared_Date {
                        Year: proto.Int32(2015),
                        Month: proto.Int32(4),
                        Day: proto.Int32(1),
                        Hours: proto.Int32(0),
                        Min: proto.Int32(0),
                        Sec: proto.Int32(0),
                    },
                    EventTimingMod: proto.Float32(-0.0833333283662796),
                }

                attBlob, err = proto.Marshal(rewardProgress)
                check(err)
                respType = int64(hsproto.PegasusUtil_RewardProgress_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_BOOSTER_TALLY {
                boosterTally := &hsproto.PegasusUtil_BoosterTallyList {}
                attBlob, err = proto.Marshal(boosterTally)
                check(err)
                respType = int64(hsproto.PegasusUtil_BoosterTallyList_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_PLAYER_RECORD {
                playerRecords := &hsproto.PegasusUtil_PlayerRecords {}
                attBlob, err = proto.Marshal(playerRecords)
                check(err)
                respType = int64(hsproto.PegasusUtil_PlayerRecords_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_GOLD_BALANCE {
                goldBalance := &hsproto.PegasusUtil_GoldBalance {
                    CappedBalance: proto.Int64(1000),
                    BonusBalance: proto.Int64(0),
                    Cap: proto.Int64(999999),
                    CapWarning: proto.Int64(999999),
                }
                attBlob, err = proto.Marshal(goldBalance)
                check(err)
                respType = int64(hsproto.PegasusUtil_GoldBalance_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_HERO_XP {
                heroXP := &hsproto.PegasusUtil_HeroXP {
                    XpInfos: make([]*hsproto.PegasusUtil_HeroXPInfo, 10),
                }
                for i := int32(2); i < 12; i++ {
                    heroXP.XpInfos[i - 2] = &hsproto.PegasusUtil_HeroXPInfo {
                        ClassId: proto.Int32(i),
                        Level: proto.Int32(60),
                        CurrXp: proto.Int64(1480),
                        MaxXp: proto.Int64(1480),
                    }
                }
                attBlob, err = proto.Marshal(heroXP)
                check(err)
                respType = int64(hsproto.PegasusUtil_HeroXP_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_CARD_BACKS {
                cardBacks := &hsproto.PegasusUtil_CardBacks {
                    DefaultCardBack: proto.Int32(13),
                    CardBacks: make([]int32, 19),
                }
                for i := int32(1); i < 20; i++ {
                    cardBacks.CardBacks[i - 1] = i
                }
                attBlob, err = proto.Marshal(cardBacks)
                check(err)
                respType = int64(hsproto.PegasusUtil_CardBacks_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_FAVORITE_HEROES {
                playerRecords := &hsproto.PegasusUtil_FavoriteHeroesResponse {}
                attBlob, err = proto.Marshal(playerRecords)
                check(err)
                respType = int64(hsproto.PegasusUtil_FavoriteHeroesResponse_PacketID_value["ID"])
            } else if requestType == hsproto.PegasusUtil_GetAccountInfo_TAVERN_BRAWL_INFO {
                tavernInfo := &hsproto.PegasusUtil_TavernBrawlInfo {}
                attBlob, err = proto.Marshal(tavernInfo)
                check(err)
                respType = int64(hsproto.PegasusUtil_TavernBrawlInfo_PacketID_value["ID"])
            }
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(respType),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: attBlob,
                        },
                    },
                },
            }
        } else if packetType == 305 {
            fmt.Println("pegasus 305 - adventure progress")
            adventureProgress := &hsproto.PegasusUtil_AdventureProgressResponse {}
            adventureData, err := proto.Marshal(adventureProgress)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(int64(hsproto.PegasusUtil_AdventureProgressResponse_PacketID_value["ID"])),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: adventureData,
                        },
                    },
                },
            }
        } else if packetType == 240 {
            fmt.Println("pegasus 240 - get options")
            clientOptions := &hsproto.PegasusUtil_ClientOptions {}
            clientData, err := proto.Marshal(clientOptions)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(int64(hsproto.PegasusUtil_ClientOptions_PacketID_value["ID"])),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: clientData,
                        },
                    },
                },
            }
        } else if packetType == 237 {
            fmt.Println("pegasus 237 - get battle pay status")
            battlePayConfig := &hsproto.PegasusUtil_BattlePayConfigResponse {
                Currency: proto.Int32(2),
                Unavailable: proto.Bool(true),
                SecsBeforeAutoCancel: proto.Int32(600),
                GoldCostArena: proto.Int64(150),
            }
            battlePayData, err := proto.Marshal(battlePayConfig)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(int64(hsproto.PegasusUtil_BattlePayConfigResponse_PacketID_value["ID"])),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: battlePayData,
                        },
                    },
                },
            }
        } else if packetType == 253 {
            fmt.Println("pegasus 253 - get achieves")
            achieves := &hsproto.PegasusUtil_Achieves {}
            achievesData, err := proto.Marshal(achieves)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(int64(hsproto.PegasusUtil_Achieves_PacketID_value["ID"])),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: achievesData,
                        },
                    },
                },
            }
        } else if int32(packetType) == hsproto.PegasusUtil_ValidateAchieve_PacketID_value["ID"] {
            fmt.Println("pegasus get achieves")
            validateAchieveReq := &hsproto.PegasusUtil_ValidateAchieve {}
            err := proto.Unmarshal(pegasusData, validateAchieveReq)
            achieves := &hsproto.PegasusUtil_ValidateAchieveResponse {
                Achieve: proto.Int32(validateAchieveReq.GetAchieve()),
            }
            achievesData, err := proto.Marshal(achieves)
            check(err)
            resp = &hsproto.BnetProtocolGameUtilities_ClientResponse {
                Attribute: []*hsproto.BnetProtocolAttribute_Attribute {
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            IntValue: proto.Int64(int64(hsproto.PegasusUtil_ValidateAchieveResponse_PacketID_value["ID"])),
                        },
                    },
                    &hsproto.BnetProtocolAttribute_Attribute {
                        Name: &name,
                        Value: &hsproto.BnetProtocolAttribute_Variant {
                            BlobValue: achievesData,
                        },
                    },
                },
            }
        } else {
            fmt.Printf("UNSOLVED packet type %d\n", packetType)
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

            fmt.Printf("%d %d %d\n",read, len(buf), idx)
			processed := session.handleRequest(buf[:idx])
            if processed < 0 {
                fmt.Printf("packet too short %d\n", processed)
                if idx != 0 {
                    //t := idx
                    //copy(buf, buf[idx:])
                    //idx = len(buf) - t
                    break
                } else {
                    buf = append(buf, make([]byte, len(buf)>>1)...)
                }
            }
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

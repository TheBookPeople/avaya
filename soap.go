package avaya

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"tbp/avaya/soap"
	"tbp/avaya/soap/CICustomerWs"
	"tbp/avaya/soap/CISkillsetWs"
	"tbp/avaya/soap/CIUtilityWs"
	"tbp/avaya/soap/CIWebCommsWs"
)

func newCIUtilityWs() *CIUtilityWs.Soap {
	return CIUtilityWs.NewSoap("", false, &soap.BasicAuth{})
}

func newCISkillset() *CISkillsetWs.Soap {
	return CISkillsetWs.NewSoap("", false, &soap.BasicAuth{})
}

func newCIWebComms() *CIWebCommsWs.Soap {
	return CIWebCommsWs.NewSoap("", false, &soap.BasicAuth{})
}

func newCICustomerWs() *CICustomerWs.Soap {
	return CICustomerWs.NewSoap("", false, &soap.BasicAuth{})
}

func anonymousLogin(ctx context.Context) (string, int64, error) {
	ciUtil := newCIUtilityWs()
	resp, err := ciUtil.GetAnonymousSessionKey(ctx, &CIUtilityWs.GetAnonymousSessionKey{})
	if err != nil {
		return "", 0, fmt.Errorf("Failed to get anonymous session key: %v", err)
	}
	anonymousID, err := strconv.Atoi(resp.GetAnonymousSessionKeyResult.AnonymousID)
	if err != nil {
		return "", 0, fmt.Errorf("Failed to convert anonymousID to int64: %v %v", resp.GetAnonymousSessionKeyResult.AnonymousID, err)
	}
	return resp.GetAnonymousSessionKeyResult.SessionKey, int64(anonymousID), nil
}

func keepAlive(ctx context.Context, sessionID string, contactID int64, isTyping bool) error {
	ciWebComms := newCIWebComms()
	_, err := ciWebComms.UpdateAliveTimeAndUpdateIsTyping(ctx, &CIWebCommsWs.UpdateAliveTimeAndUpdateIsTyping{
		ContactID:  contactID,
		SessionKey: sessionID,
		IsTyping:   isTyping,
	})

	return err
}

func customerID(ctx context.Context, sessionID string, anonymousID int64, email string) (int64, error) {

	ciUtil := newCIUtilityWs()
	resp, err := ciUtil.GetAndUpdateAnonymousCustomerID(ctx, &CIUtilityWs.GetAndUpdateAnonymousCustomerID{
		LoginResult: &CIUtilityWs.AnonymousLoginResult{
			AnonymousID: anonymousID,
			SessionKey:  sessionID,
		},
		EmailAddress: email,
		ThisCustomer: &CIUtilityWs.CICustomerReadType{
			AddressList: &CIUtilityWs.ArrayOfCIAddressReadType{
				CIAddressReadType: []*CIUtilityWs.CIAddressReadType{
					&CIUtilityWs.CIAddressReadType{},
				},
			},
		},
	})
	if err != nil {
		return 0, err
	}
	return resp.GetAndUpdateAnonymousCustomerIDResult, nil
}

func IsSkillsetInService(ctx context.Context, skillsetName string) (bool, error) {
	resp, err := newCISkillset().IsSkillsetNameInService(ctx, &CISkillsetWs.IsSkillsetNameInService{
		SkillsetName: skillsetName,
	})
	if err != nil {
		return false, err
	}
	return resp.IsSkillsetNameInServiceResult, nil
}

type Skillset struct {
	ID   int64
	Name string
}

func skillset(ctx context.Context, sessionID, name string) (*Skillset, error) {
	ciSkillset := newCISkillset()
	resp, err := ciSkillset.GetSkillsetByName(ctx, &CISkillsetWs.GetSkillsetByName{
		SessionKey:   sessionID,
		SkillsetName: name,
	})

	if err != nil {
		return nil, err
	}

	return &Skillset{
		ID:   resp.GetSkillsetByNameResult.ID,
		Name: resp.GetSkillsetByNameResult.Name,
	}, nil
}

func requestChat(ctx context.Context, customerID int64, sessionID string, skillsetID int64) (int64, error) {
	ciCustomerWs := newCICustomerWs()

	resp, err := ciCustomerWs.RequestTextChat(ctx, &CICustomerWs.RequestTextChat{
		CustID:     customerID,
		SessionKey: sessionID,
		NewContact: &CICustomerWs.CIContactWriteType{
			SkillsetID: skillsetID,
		},
	})

	if err != nil {
		return 0, err
	}

	return resp.RequestTextChatResult, nil
}

func readMessages(ctx context.Context, sessionKey string, contactID int64, isWriting bool, lastReadTime int64) (*CIWebCommsWs.CIMultipleChatMessageReadType, error) {
	ciWebComms := CIWebCommsWs.NewSoap("", false, &soap.BasicAuth{})
	resp, err := ciWebComms.ReadChatMessage(ctx, &CIWebCommsWs.ReadChatMessage{
		ContactID: contactID,
		IsWriting: isWriting,
		LastReadTime: &CIWebCommsWs.CIDateTime{
			Milliseconds: lastReadTime,
		},
		SessionKey: sessionKey,
	})

	if err != nil {
		return nil, err
	}

	result := resp.ReadChatMessageResult
	log.Println(resp)
	return result, nil
}

const (
	fromCustomer         = CIWebCommsWs.CIChatMessageTypeChatMessagefromCustomer
	customerDisconnected = CIWebCommsWs.CIChatMessageTypeSessionDisconnectedbyCustomer
	agentDisconnected    = CIWebCommsWs.CIChatMessageTypeSessionDisconnectedbyAgent
)

func writeMessage(ctx context.Context, sessionKey string, contactID int64, message string, msgType CIWebCommsWs.CIChatMessageType) error {

	ciWebComms := CIWebCommsWs.NewSoap("", false, &soap.BasicAuth{})
	resp, err := ciWebComms.WriteChatMessage(ctx, &CIWebCommsWs.WriteChatMessage{
		ContactID:       contactID,
		Message:         message,
		SessionKey:      sessionKey,
		ChatMessageType: &msgType,
	})

	if err != nil {
		return err
	}

	fmt.Printf("###\nchat message result:%d\n###", resp.WriteChatMessageResult)
	return nil
}

func AbandonQue(ctx context.Context, sessionKey string, contactID int64, reason string) error {
	ciWebComms := newCIWebComms()
	_, err := ciWebComms.AbandonQueuingWebCommsContact(ctx, &CIWebCommsWs.AbandonQueuingWebCommsContact{
		SessionKey:     sessionKey,
		ContactID:      contactID,
		ClosureComment: reason,
	})
	return err
}

func EndSession(ctx context.Context, sessionKey string, contactID int64) error {
	ciUtil := newCIUtilityWs()
	_, err := ciUtil.CustomerLogoffByContactID(ctx, &CIUtilityWs.CustomerLogoffByContactID{
		SessionKey: sessionKey,
		ContactID:  contactID,
		// FIXME: Does username do anything?
	})
	return err
}

/*
web_1          | <env:Envelope xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:tns="http://webservices.ci.ccmm.applications.nortel.com" xmlns:env="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ins0="http://datatypes.ci.ccmm.applications.nortel.com">
web_1          |   <env:Body>
web_1          |     <tns:RequestTextChat>
web_1          |       <tns:custID>198853</tns:custID>
web_1          |       <tns:sessionKey>4145hiDT00</tns:sessionKey>
web_1          |       <tns:newContact>
web_1          |         <ins0:skillsetID>11</ins0:skillsetID>
web_1          |       </tns:newContact>
web_1          |     </tns:RequestTextChat>
web_1          |   </env:Body>
web_1          | </env:Envelope>
*/
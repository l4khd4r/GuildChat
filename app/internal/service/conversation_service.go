package service

import (
	"context"
	"errors"

	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)


type ConversationService struct {
	conversationRepo *repository.ConversationRepository
}


func NewConversationService(conversationRepo *repository.ConversationRepository) *ConversationService {
	return &ConversationService{
		conversationRepo : conversationRepo,
	}
}



func (s *ConversationService) GetOrCreateDM(ctx context.Context , userID1 int64 , userID2 int64) (*model.Conversation, error) {
	if userID1 == userID2 {
		return nil , errors.New(model.ErrorCannotCreateDMWithSelf)
	}


	conversation , err := s.conversationRepo.GetDM(ctx, userID1 , userID2)


	if err == nil {
		return conversation , nil
	}
	// here we create a new DM conversation


	if errors.Is(err , repository.ErrNotFound) {
		return nil , errors.New(model.ErrorConversationNotFound)
	}



	conversation , err = s.conversationRepo.CreateConversation(ctx,
		 model.ConversationDM ,
		 nil ,   // no name for DM
		 userID1 , )


	if err != nil {
		return nil , err
	}




	err = s.conversationRepo.AddMember(ctx , conversation.ID , userID1 , model.MemberOwner)
	if err != nil {
		return nil , err
	}

	err = s.conversationRepo.AddMember(ctx , conversation.ID , userID2 , model.MemberMember)
	if err != nil {
		return nil , err
	}

	return conversation  , nil
}

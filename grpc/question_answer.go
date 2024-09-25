package grpc

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/neo4j2"
)

func (h *Handler) SaveQuestionAndAnswer(ctx context.Context, req *connect.Request[api.SaveQuestionAndAnswerRequest]) (*connect.Response[api.SaveQuestionAndAnswerResponse], error) {
	for _, v := range req.Msg.QuestionAndAnswer {
		question := v.Question
		answer := v.Answer
		embedding, err := ai.EmbedQuestion(h.openaiClient, question)
		if err != nil {
			log.Printf("Error embedding question: %v", err)
		}
		err = neo4j2.CreateQuestionAndAnswers(ctx, h.neo4jDriver, question, embedding, []string{answer})
		if err != nil {
			log.Printf("Error saving question and answer to database: %v", err)
			return &connect.Response[api.SaveQuestionAndAnswerResponse]{Msg: &api.SaveQuestionAndAnswerResponse{}}, err
		}
	}

	return &connect.Response[api.SaveQuestionAndAnswerResponse]{Msg: &api.SaveQuestionAndAnswerResponse{}}, nil
}

func (h *Handler) PageQuestions(ctx context.Context, req *connect.Request[api.PageQuestionsRequest]) (*connect.Response[api.PageQuestionsResponse], error) {
	dbQuestions, err := neo4j2.PageQuestions(ctx, h.neo4jDriver, int(req.Msg.Page), int(req.Msg.PageSize))
	if err != nil {
		log.Printf("Error paging questions: %v", err)
		return &connect.Response[api.PageQuestionsResponse]{Msg: &api.PageQuestionsResponse{}}, err
	}

	questions := make([]*api.Question, 0)
	for _, v := range dbQuestions {
		questions = append(questions, &api.Question{
			Id:        v.ID,
			Text:      v.Text,
			CreatedAt: v.CreatedAt,
		})
	}

	return &connect.Response[api.PageQuestionsResponse]{Msg: &api.PageQuestionsResponse{Questions: questions}}, nil

}

func (h *Handler) AnswerQuestion(ctx context.Context, req *connect.Request[api.AnswerQuestionRequest]) (*connect.Response[api.AnswerQuestionResponse], error) {
	res := &api.AnswerQuestionResponse{
		Answers:          []*api.Answer{},
		SimilarQuestions: []*api.Question{},
	}
	userEmbedding, err := ai.EmbedQuestion(h.openaiClient, req.Msg.Questions)
	if err != nil {
		return nil, err
	}
	if len(userEmbedding) == 0 {
		return nil, fmt.Errorf("Failed to embed question, zero length vector returned")
	}

	similarQuestions, err := neo4j2.VectorSearchQuestions(ctx, h.neo4jDriver, userEmbedding, 3)
	if err != nil {
		return nil,
			err
	}

	topQuestionIds := make([]string, 0)
	for _, v := range similarQuestions {
		topQuestionIds = append(topQuestionIds, v.ID)
	}

	topAnswers, err := neo4j2.GetTopAnswersForQuestions(ctx, h.neo4jDriver, topQuestionIds)
	if err != nil {
		return nil, err
	}
	finalAnswer := ""
	if len(topAnswers) > 0 {
		finalAnswer = topAnswers[0].Answer
	}
	// generate the final answer using ai
	if req.Msg.UseAi {
		finalAnswer, err = ai.GenerateFinalAnswer(h.instructorClient, req.Msg.Questions, topAnswers)
		if err !=
			nil {
			return nil, err
		}
	}
	res.Answers = append(res.Answers, &api.Answer{
		Answer:   finalAnswer,
		Question: req.Msg.Questions,
	})
	for _, v := range similarQuestions {
		res.SimilarQuestions = append(res.SimilarQuestions, &api.Question{
			Id:        v.ID,
			Text:      v.Text,
			CreatedAt: v.CreatedAt,
		})
	}

	return &connect.Response[api.AnswerQuestionResponse]{
		Msg: res,
	}, nil
}

func (h *Handler) SaveConversationSummary(ctx context.Context, req *connect.Request[api.SaveConversationSummaryRequest]) (*connect.Response[api.SaveConversationSummaryResponse], error) {
	conversationSummary := req.Msg.ConversationSummary
	dateISO := req.Msg.DateIso
	err := neo4j2.SaveConversationSummary(ctx, h.neo4jDriver, conversationSummary, dateISO)
	if err != nil {
		log.Printf("Error saving conversation summary to database: %v", err)
		return &connect.Response[api.SaveConversationSummaryResponse]{Msg: &api.SaveConversationSummaryResponse{
			Ok: false}}, err
	}
	// questionsAndAnswers, err := h.
	// 	generateQuestionsAndAnswers(ctx,
	// 		conversationSummary,
	// 	)
	// if err != nil {
	// 	log.
	// 		Printf("Error generating questions and answers from conversation summary: %v",

	// 			err)
	// 	return &connect.Response[api.SaveToKnowledgeBaseResponse]{Msg: &api.SaveToKnowledgeBaseResponse{Ok: false}}, err
	// }

	// for _, qa := range questionsAndAnswers {
	// 	embedding, err := ai.EmbedQuestion(h.openaiClient, qa.Question)
	// 	if err != nil {
	// 		log.Printf("Error embedding question: %v", err)
	// 	}
	// 	err = neo4j2.CreateQuestionAndAnswers(ctx,
	// 		h.neo4jDriver,
	// 		qa.Question,
	// 		embedding,
	// 		qa.Answers)
	// 	if err != nil {
	// 		log.Printf("Error saving generated question and answers to Neo4j: %v",

	// 			err)
	// 		return &connect.Response[api.SaveToKnowledgeBaseResponse]{Msg: &api.SaveToKnowledgeBaseResponse{Ok: false}}, err
	// 	}
	// }
	return &connect.Response[api.SaveConversationSummaryResponse]{Msg: &api.SaveConversationSummaryResponse{Ok: true}}, nil
}

// func (h *Handler) generateQuestionsAndAnswers(ctx context.Context, conversationSummary string) ([]QuestionAndAnswers, error) {
// 	// Define the structure for AI output
// 	type AiOutput struct {
// 		QuestionsAndAnswers []QuestionAndAnswers `json:"questions_and_answers"`
// 	}

// 	// Prepare the prompt
// 	prompt := "Generate questions and answers so that my bot build a solid knowledge based on the following conversation summary:\n\n" + conversationSummary

// 	// Initialize the AI output
// 	var aiOut AiOutput

// 	// Call OpenAI API to generate questions and answers
// 	_, err := h.instructorClient.CreateChatCompletion(
// 		ctx,
// 		openai.ChatCompletionRequest{
// 			Model: openai.GPT4o, // Adjust the model as per requirements
// 			Messages: []openai.ChatCompletionMessage{
// 				{
// 					Role:    openai.ChatMessageRoleUser,
// 					Content: prompt,
// 				},
// 			},
// 		},
// 		&aiOut,
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to generate questions and answers: %v", err)
// 	}

// 	return aiOut.QuestionsAndAnswers, nil
// }

// QuestionAndAnswers struct to hold generated questions and answers
// type QuestionAndAnswers struct {
// 	Question string   `json:"question"`
// 	Answers  []string `json:"answers"`
// }

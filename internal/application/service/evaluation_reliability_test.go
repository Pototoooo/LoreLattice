package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/Pototoooo/lorelattice/internal/types/interfaces"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

func TestEvaluationReliabilityMaximumPID(t *testing.T) {
	got := getPassageList([]*types.QAPair{{PIDs: []int{1, 4}, Passages: []string{"first", "last"}}})
	require.Len(t, got, 5)
	require.Equal(t, "last", got[4])
}
func TestEvaluationReliabilityEmptyPassages(t *testing.T) {
	require.Empty(t, getPassageList(nil))
	require.Equal(t, []string{"zero"}, getPassageList([]*types.QAPair{{PIDs: []int{0}, Passages: []string{"zero"}}}))
}
func reliabilityWriteDataset(t *testing.T, dir, question string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, parquet.WriteFile(filepath.Join(dir, "queries.parquet"), []TextInfo{{ID: 1, Text: question}}))
	require.NoError(t, parquet.WriteFile(filepath.Join(dir, "corpus.parquet"), []TextInfo{{ID: 1, Text: "passage"}}))
	require.NoError(t, parquet.WriteFile(filepath.Join(dir, "answers.parquet"), []TextInfo{{ID: 1, Text: "answer"}}))
	require.NoError(t, parquet.WriteFile(filepath.Join(dir, "qrels.parquet"), []RelsInfo{{QID: 1, PID: 1}}))
	require.NoError(t, parquet.WriteFile(filepath.Join(dir, "qas.parquet"), []QaInfo{{QID: 1, AID: 1}}))
}
func TestEvaluationReliabilityDatasetSelection(t *testing.T) {
	t.Chdir(t.TempDir())
	reliabilityWriteDataset(t, "dataset/samples", "default question")
	reliabilityWriteDataset(t, "dataset/custom_v1", "custom question")
	svc := NewDatasetService()
	for _, id := range []string{"", "default", "samples"} {
		pairs, err := svc.GetDatasetByID(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, "default question", pairs[0].Question)
	}
	pairs, err := svc.GetDatasetByID(context.Background(), "custom_v1")
	require.NoError(t, err)
	require.Equal(t, "custom question", pairs[0].Question)
}
func TestEvaluationReliabilityDatasetErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	reliabilityWriteDataset(t, "dataset/samples", "default")
	for _, id := range []string{"missing", "../samples", "/tmp/samples", "a/b", "a\\b", ".", ".."} {
		_, err := NewDatasetService().GetDatasetByID(context.Background(), id)
		require.Error(t, err, "id=%q", id)
	}
	require.NoError(t, os.MkdirAll("dataset/broken", 0755))
	require.NotPanics(t, func() {
		_, err := NewDatasetService().GetDatasetByID(context.Background(), "broken")
		require.Error(t, err)
	})
}

type reliabilityDataset struct{ interfaces.DatasetService }

func (*reliabilityDataset) GetDatasetByID(context.Context, string) ([]*types.QAPair, error) {
	return []*types.QAPair{{QID: 1, PIDs: []int{1}, Passages: []string{"source"}, Question: "question", Answer: "answer"}}, nil
}

type reliabilityKnowledge struct {
	interfaces.KnowledgeService
	indexed, asynchronous, deleted bool
	importErr                      error
}

func (s *reliabilityKnowledge) CreateKnowledgeFromPassage(context.Context, string, []string, string) (*types.Knowledge, error) {
	s.asynchronous = true
	return &types.Knowledge{ID: "knowledge"}, nil
}
func (s *reliabilityKnowledge) CreateKnowledgeFromPassageSync(context.Context, string, []string, string) (*types.Knowledge, error) {
	if s.importErr != nil {
		return nil, s.importErr
	}
	s.indexed = true
	return &types.Knowledge{ID: "knowledge"}, nil
}
func (s *reliabilityKnowledge) DeleteKnowledge(context.Context, string) error {
	s.deleted = true
	return nil
}

type reliabilityKB struct {
	interfaces.KnowledgeBaseService
	deleted bool
}

func (s *reliabilityKB) DeleteKnowledgeBase(context.Context, string) error {
	s.deleted = true
	return nil
}

var reliabilityStop = errors.New("stop after checking index readiness")

type reliabilitySession struct {
	interfaces.SessionService
	knowledge *reliabilityKnowledge
	called    bool
}

func (s *reliabilitySession) KnowledgeQAByEvent(context.Context, *types.ChatManage, []types.EventType) error {
	s.called = true
	if !s.knowledge.indexed {
		return errors.New("query started before indexing")
	}
	return reliabilityStop
}
func TestEvaluationReliabilityWaitForIndex(t *testing.T) {
	knowledge := &reliabilityKnowledge{}
	kb := &reliabilityKB{}
	session := &reliabilitySession{knowledge: knowledge}
	svc := &EvaluationService{dataset: &reliabilityDataset{}, knowledgeService: knowledge, knowledgeBaseService: kb, sessionService: session, evaluationMemoryStorage: newEvaluationMemoryStorage()}
	detail := &types.EvaluationDetail{Task: &types.EvaluationTask{ID: "test", DatasetID: "default"}, Params: &types.ChatManage{}}
	svc.evaluationMemoryStorage.register(detail)
	err := svc.EvalDataset(context.Background(), detail, "kb")
	require.ErrorIs(t, err, reliabilityStop)
	require.True(t, knowledge.indexed)
	require.False(t, knowledge.asynchronous)
	require.True(t, session.called)
	require.True(t, knowledge.deleted)
	require.True(t, kb.deleted)
}
func TestEvaluationReliabilityIndexFailureStopsQuery(t *testing.T) {
	importErr := errors.New("index failed")
	knowledge := &reliabilityKnowledge{importErr: importErr}
	session := &reliabilitySession{knowledge: knowledge}
	svc := &EvaluationService{dataset: &reliabilityDataset{}, knowledgeService: knowledge, knowledgeBaseService: &reliabilityKB{}, sessionService: session, evaluationMemoryStorage: newEvaluationMemoryStorage()}
	detail := &types.EvaluationDetail{Task: &types.EvaluationTask{ID: "test", DatasetID: "default"}, Params: &types.ChatManage{}}
	svc.evaluationMemoryStorage.register(detail)
	require.ErrorIs(t, svc.EvalDataset(context.Background(), detail, "kb"), importErr)
	require.False(t, session.called)
}

func TestEvaluationReliabilityDatasetSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	reliabilityWriteDataset(t, outside, "outside")
	t.Chdir(t.TempDir())
	reliabilityWriteDataset(t, "dataset/samples", "default")
	require.NoError(t, os.Symlink(outside, "dataset/escape"))
	_, err := NewDatasetService().GetDatasetByID(context.Background(), "escape")
	require.Error(t, err)
}
func TestEvaluationReliabilityCorruptDataset(t *testing.T) {
	t.Chdir(t.TempDir())
	reliabilityWriteDataset(t, "dataset/broken", "question")
	require.NoError(t, os.WriteFile("dataset/broken/queries.parquet", []byte("invalid parquet"), 0600))
	require.NotPanics(t, func() {
		_, err := NewDatasetService().GetDatasetByID(context.Background(), "broken")
		require.Error(t, err)
	})
}

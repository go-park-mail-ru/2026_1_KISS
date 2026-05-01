package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/ctxutil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

var errNotSupported = errors.New("operation not supported by adapter")

type NotebookAdapter struct {
	client pb.NotebookServiceClient
}

func NewNotebookAdapter(client pb.NotebookServiceClient) *NotebookAdapter {
	return &NotebookAdapter{client: client}
}

func (a *NotebookAdapter) Create(_ context.Context, _ *domain.Notebook) (int64, error) {
	return 0, errNotSupported
}

func (a *NotebookAdapter) GetByID(ctx context.Context, id int64) (*domain.Notebook, error) {
	resp, err := a.client.GetByID(ctx, &pb.GetNotebookRequest{NotebookId: id, UserId: ctxutil.UserIDFromContext(ctx)})
	if err != nil {
		return nil, err
	}
	return protoToNotebook(resp.GetNotebook()), nil
}

func (a *NotebookAdapter) GetByOwnerID(_ context.Context, _ int64, _, _ int, _ string) ([]domain.Notebook, error) {
	return nil, errNotSupported
}

func (a *NotebookAdapter) Update(_ context.Context, _ *domain.Notebook) error {
	return errNotSupported
}

func (a *NotebookAdapter) Delete(_ context.Context, _ int64) error {
	return errNotSupported
}

func (a *NotebookAdapter) CountByOwnerID(_ context.Context, _ int64, _ string) (int, error) {
	return 0, errNotSupported
}

func (a *NotebookAdapter) ListAll(_ context.Context, _, _ int, _ string) ([]domain.Notebook, error) {
	return nil, errNotSupported
}

func (a *NotebookAdapter) CountAll(_ context.Context, _ string) (int, error) {
	return 0, errNotSupported
}

func (a *NotebookAdapter) GetSharedWithUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notebook, error) {
	resp, err := a.client.ListSharedWithUser(ctx, &pb.ListSharedWithUserRequest{
		UserId: userID,
		Limit:  int32(limit),  //nolint:gosec // pagination limit fits int32
		Offset: int32(offset), //nolint:gosec // pagination offset fits int32
	})
	if err != nil {
		return nil, err
	}
	notebooks := make([]domain.Notebook, len(resp.GetNotebooks()))
	for i, nb := range resp.GetNotebooks() {
		n := protoToNotebook(nb)
		if n != nil {
			notebooks[i] = *n
		}
	}
	return notebooks, nil
}

func (a *NotebookAdapter) SetAllPrivateByOwner(_ context.Context, _ int64) error {
	return errNotSupported
}

func (a *NotebookAdapter) CountSharedWithUser(ctx context.Context, userID int64) (int, error) {
	resp, err := a.client.ListSharedWithUser(ctx, &pb.ListSharedWithUserRequest{
		UserId: userID,
		Limit:  0,
		Offset: 0,
	})
	if err != nil {
		return 0, err
	}
	return int(resp.GetTotal()), nil
}

type BlockAdapter struct {
	client pb.NotebookServiceClient
}

func NewBlockAdapter(client pb.NotebookServiceClient) *BlockAdapter {
	return &BlockAdapter{client: client}
}

func (a *BlockAdapter) Create(_ context.Context, _ *domain.Block) (int64, error) {
	return 0, errNotSupported
}

func (a *BlockAdapter) GetByID(_ context.Context, _ int64) (*domain.Block, error) {
	return nil, errNotSupported
}

func (a *BlockAdapter) GetByNotebookID(ctx context.Context, notebookID int64) ([]domain.Block, error) {
	resp, err := a.client.GetBlocksByNotebookID(ctx, &pb.GetBlocksRequest{NotebookId: notebookID, UserId: ctxutil.UserIDFromContext(ctx)})
	if err != nil {
		return nil, err
	}
	blocks := make([]domain.Block, len(resp.GetBlocks()))
	for i, b := range resp.GetBlocks() {
		blk := domain.Block{
			ID:         b.GetId(),
			NotebookID: b.GetNotebookId(),
			Type:       b.GetType(),
			Language:   b.GetLanguage(),
			Content:    b.GetContent(),
			Position:   int(b.GetPosition()),
			CreatedAt:  time.Unix(b.GetCreatedAt(), 0),
			UpdatedAt:  time.Unix(b.GetUpdatedAt(), 0),
		}
		if b.ExecutionCount != nil {
			v := int(b.GetExecutionCount())
			blk.ExecutionCount = &v
		}
		blocks[i] = blk
	}
	return blocks, nil
}

func (a *BlockAdapter) Update(_ context.Context, _ *domain.Block) error {
	return errNotSupported
}

func (a *BlockAdapter) Delete(_ context.Context, _ int64) error {
	return errNotSupported
}

func (a *BlockAdapter) SaveOutputs(_ context.Context, _ int64, _ []domain.BlockOutput) error {
	return errNotSupported
}

func (a *BlockAdapter) GetOutputsByBlockIDs(_ context.Context, _ []int64) (map[int64][]domain.BlockOutput, error) {
	return nil, errNotSupported
}

func (a *BlockAdapter) ReorderBlocks(_ context.Context, _ int64, _ []int64) error {
	return errNotSupported
}

func (a *BlockAdapter) CreateBatch(_ context.Context, _ []domain.Block) ([]int64, error) {
	return nil, errNotSupported
}

func (a *BlockAdapter) CountByOwnerID(_ context.Context, _ int64) (int64, error) {
	return 0, errNotSupported
}

func (a *BlockAdapter) SumExecutionsByOwnerID(_ context.Context, _ int64) (int64, error) {
	return 0, errNotSupported
}

func (a *BlockAdapter) IncrementExecutionCount(_ context.Context, _ int64) error {
	return errNotSupported
}

func (a *BlockAdapter) CountExecutionsByOwnerByDay(_ context.Context, _ int64, _ time.Time) ([]domain.DayCount, error) {
	return nil, errNotSupported
}

func protoToNotebook(info *pb.NotebookInfo) *domain.Notebook {
	if info == nil {
		return nil
	}
	nb := &domain.Notebook{
		ID:        info.GetId(),
		OwnerID:   info.GetOwnerId(),
		Title:     info.GetTitle(),
		IsPublic:  info.GetIsPublic(),
		CreatedAt: time.Unix(info.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(info.GetUpdatedAt(), 0),
	}
	if len(info.GetBlocks()) > 0 {
		nb.Blocks = make([]domain.Block, len(info.GetBlocks()))
		for i, b := range info.GetBlocks() {
			blk := domain.Block{
				ID:         b.GetId(),
				NotebookID: b.GetNotebookId(),
				Type:       b.GetType(),
				Language:   b.GetLanguage(),
				Content:    b.GetContent(),
				Position:   int(b.GetPosition()),
				CreatedAt:  time.Unix(b.GetCreatedAt(), 0),
				UpdatedAt:  time.Unix(b.GetUpdatedAt(), 0),
			}
			if b.ExecutionCount != nil {
				v := int(b.GetExecutionCount())
				blk.ExecutionCount = &v
			}
			nb.Blocks[i] = blk
		}
	}
	return nb
}

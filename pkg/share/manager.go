package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Settings struct {
	TableName struct {
		Owner  string `cfg:"owner" default:"owners"`
		Policy string `cfg:"policy" default:"policies"`
		Share  string `cfg:"share" default:"shares"`
	} `cfg:"table_name"`
}

type shareManagerAppctxKey string

type Share struct {
	db_repo.Model
	EntityId   uint
	EntityType string
	OwnerId    uint
	PolicyId   string
}

func (s *Share) GetPolicyId() string {
	return s.PolicyId
}

type shareManager struct {
	orm      db_repo.Remote
	logger   log.Logger
	settings Settings
}

func ProvideShareManager(ctx context.Context, config cfg.Config, logger log.Logger) (*shareManager, error) {
	return appctx.Provide(ctx, shareManagerAppctxKey("shareManager"), func() (*shareManager, error) {
		return NewShareManager(config, logger)
	})
}

func ProvideShareManagerFactory(ctx context.Context, config cfg.Config, logger log.Logger) (func(db_repo.Remote) *shareManager, error) {
	return appctx.Provide(ctx, shareManagerAppctxKey("shareManagerFactory"), func() (func(db_repo.Remote) *shareManager, error) {
		return NewShareManagerFactory(config, logger), nil
	})
}

func NewShareManager(config cfg.Config, logger log.Logger) (*shareManager, error) {
	orm, err := db_repo.NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return NewShareManagerFactory(config, logger)(orm), nil
}

func NewShareManagerFactory(config cfg.Config, logger log.Logger) func(db_repo.Remote) *shareManager {
	settings := Settings{}
	config.UnmarshalKey("share", &settings)

	return func(remote db_repo.Remote) *shareManager {
		return &shareManager{
			logger:   logger.WithChannel("share_manager"),
			orm:      remote,
			settings: settings,
		}
	}
}

func (m *shareManager) SetupShareTable() error {
	tn := m.settings.TableName

	scope := m.orm.NewScope(Share{})
	if scope.Dialect().HasTable(tn.Share) {
		return nil
	}

	_, err := m.orm.CommonDB().Exec(fmt.Sprintf(`
		create table %s
		(
			id               int unsigned auto_increment primary key,
			entity_id        int unsigned not null,
			entity_type		 varchar(255) not null,
			owner_id       	 int unsigned not null,
			policy_id        varchar(255) not null,
			created_at       timestamp    not null,
			updated_at       timestamp    not null,

			constraint shares_owner_id_fk foreign key (owner_id) references %s (id),
			constraint shares_policy_id_fk foreign key (policy_id) references %s (id)
		)
	`, tn.Share, tn.Owner, tn.Policy))
	if err != nil {
		return fmt.Errorf("could not create table %s: %w", tn.Share, err)
	}

	return nil
}

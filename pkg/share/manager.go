package share

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
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
	orm      *gorm.DB
	logger   log.Logger
	settings Settings
}

func ProvideShareManager(ctx context.Context, config cfg.Config, logger log.Logger) (*shareManager, error) {
	return appctx.Provide(ctx, shareManagerAppctxKey("shareManager"), func() (*shareManager, error) {
		return NewShareManager(config, logger)
	})
}

func NewShareManager(config cfg.Config, logger log.Logger) (*shareManager, error) {
	orm, err := db_repo.NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	settings := Settings{}
	config.UnmarshalKey("share", &settings)

	return &shareManager{
		logger:   logger.WithChannel("share_manager"),
		orm:      orm,
		settings: settings,
	}, nil
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

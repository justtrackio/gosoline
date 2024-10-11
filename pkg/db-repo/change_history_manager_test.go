package db_repo_test

import (
	"testing"

	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

type changeHistoryManagerTestSuite struct {
	suite.Suite

	logger   *logMocks.Logger
	dbClient goSqlMock.Sqlmock
	manager  *db_repo.ChangeHistoryManager
}

type testModel struct {
	db_repo.Model
	Value string
}

func TestRunChangeHistoryManagerTestSuite(t *testing.T) {
	suite.Run(t, new(changeHistoryManagerTestSuite))
}

func (s *changeHistoryManagerTestSuite) SetupTest() {
	s.logger = logMocks.NewLogger(s.T())
	s.logger.EXPECT().WithChannel("change_history_manager").Return(s.logger)

	db, clientMock, err := goSqlMock.New()
	s.Require().NoError(err)

	orm, err := db_repo.NewOrmWithInterfaces(db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	s.Require().NoError(err)

	s.dbClient = clientMock
	s.manager = db_repo.NewChangeHistoryManagerWithInterfaces(s.logger, orm, &db_repo.ChangeHistoryManagerSettings{
		TableSuffix: "history",
	})
}

func (s *changeHistoryManagerTestSuite) TearDownTest() {
	if err := s.dbClient.ExpectationsWereMet(); err != nil {
		s.T().Errorf("there were unfulfilled expectations: %s", err)
	}
}

func (s *changeHistoryManagerTestSuite) TestRunMigration_Disabled() {
	s.logger.EXPECT().Info("creating change history setup").Once()
	s.logger.EXPECT().Info("planned schema change: CREATE TABLE `test_models_history` (`change_history_action`  VARCHAR(8) NOT NULL DEFAULT 'insert',`change_history_action_at` DATETIME NULL DEFAULT CURRENT_TIMESTAMP,`change_history_revision` int ,`change_history_author_id` int,`id` int unsigned ,`updated_at` datetime,`created_at` datetime,`value` varchar(255), PRIMARY KEY (`change_history_revision`,`id`))").Once() //nolint:lll // test scenario
	s.logger.EXPECT().Info("planned schema change: DROP TRIGGER IF EXISTS test_models_ai").Once()
	s.logger.EXPECT().Info("planned schema change: DROP TRIGGER IF EXISTS test_models_au").Once()
	s.logger.EXPECT().Info("planned schema change: DROP TRIGGER IF EXISTS test_models_bd").Once()
	s.logger.EXPECT().Info("planned schema change: DROP TRIGGER IF EXISTS test_models_history_revai").Once()
	s.logger.EXPECT().Info("planned schema change: CREATE TRIGGER test_models_ai AFTER INSERT ON `test_models` FOR EACH ROW \n\t\tINSERT INTO `test_models_history` (change_history_action,change_history_revision,change_history_action_at,`id`,`updated_at`,`created_at`,`value`,`change_history_author_id`) \n\t\t\tSELECT 'insert', NULL, NOW(), d.`id`, d.`updated_at`, d.`created_at`, d.`value`, @change_history_author_id \n\t\t\tFROM `test_models` AS d WHERE d.`id` = NEW.`id`").Once()                                                                                                                       //nolint:lll // test scenario
	s.logger.EXPECT().Info("planned schema change: CREATE TRIGGER test_models_au AFTER UPDATE ON `test_models` FOR EACH ROW \n\t\tINSERT INTO `test_models_history` (change_history_action,change_history_revision,change_history_action_at,`id`,`updated_at`,`created_at`,`value`,`change_history_author_id`) \n\t\t\tSELECT 'update', NULL, NOW(), d.`id`, d.`updated_at`, d.`created_at`, d.`value`, @change_history_author_id \n\t\t\tFROM `test_models` AS d WHERE d.`id` = NEW.`id` AND (NOT (OLD.`id` <=> NEW.`id`) OR NOT (OLD.`created_at` <=> NEW.`created_at`) OR NOT (OLD.`value` <=> NEW.`value`))").Once() //nolint:lll // test scenario
	s.logger.EXPECT().Info("planned schema change: CREATE TRIGGER test_models_bd BEFORE DELETE ON `test_models` FOR EACH ROW \n\t\tINSERT INTO `test_models_history` (change_history_action,change_history_revision,change_history_action_at,`id`,`updated_at`,`created_at`,`value`,`change_history_author_id`) \n\t\t\tSELECT 'delete', NULL, NOW(), d.`id`, d.`updated_at`, d.`created_at`, d.`value`, @change_history_author_id \n\t\t\tFROM `test_models` AS d WHERE d.`id` = OLD.`id`").Once()                                                                                                                      //nolint:lll // test scenario
	s.logger.EXPECT().Info("planned schema change: CREATE TRIGGER test_models_history_revai BEFORE INSERT ON `test_models_history` FOR EACH ROW SET NEW.change_history_revision = (SELECT IFNULL(MAX(d.change_history_revision), 0) + 1 FROM `test_models_history` as d WHERE d.`id` = NEW.`id`);").Once()                                                                                                                                                                                                                                                                                                               //nolint:lll // test scenario
	s.logger.EXPECT().Info("change history migration is disabled, please apply the changes manually").Once()

	results := goSqlMock.NewRows([]string{"name"}).AddRow("test_models")
	s.dbClient.ExpectQuery("SHOW TABLES FROM `` WHERE `Tables_in_` = \\?").WithArgs("test_models").WillReturnRows(results)

	results = goSqlMock.NewRows([]string{"name"})
	s.dbClient.ExpectQuery("SHOW TABLES FROM `` WHERE `Tables_in_` = \\?").WithArgs("test_models_history").WillReturnRows(results)

	err := s.manager.RunMigration(&testModel{})
	s.Require().Error(err)
	s.Equal("cannot execute change history migration: missing schema migrations (disabled)", err.Error())
}

package main

import (
	"time"

	"go.uber.org/zap/zapcore"
)

type Tenant struct {
	Url  string `csv:"url"`
	User string `csv:"user"`
	Pass string `csv:"pass"`
}

type (
	CLICommand struct {
		Command string `json:"command"`
	}
)

type Software struct {
	Name         string
	Version      string
	Url          string
	Id           string
	CreationTime time.Time
}

type SoftwareMigration struct {
	Software     Software
	IsOldPackage bool
	Migrated     bool
}

type MigrationAudit struct {
	ConnectionProbeSucceeded bool
	MigratedEntities         []MigrationAuditEntry
	NonMigratedEntities      []NonMigrationAuditEntry
}

type MigrationAuditEntry struct {
	SoftwareName           string
	SoftwareVersion        string
	OldSoftwareMoId        string
	NewSoftwareMoId        string
	NewSoftwareVersionMoId string
}

type NonMigrationAuditEntry struct {
	SoftwareName    string
	SoftwareVersion string
	SoftwareUrl     string
	Reason          string
}

type MigrationJob struct {
	Id        string
	StartTime time.Time
	Status    string
	Tenants   []Tenant
	ErrorLog  []map[string]string
	WarnLog   []map[string]string
	Audit     map[string]*MigrationAudit
}

func (job *MigrationJob) logError(msg string, fields []zapcore.Field) {
	m := make(map[string]string, 0)
	m["msg"] = msg
	for _, element := range fields {
		m[element.Key] = element.String
	}
	job.ErrorLog = append(job.ErrorLog, m)
}

func (job *MigrationJob) logWarning(msg string, fields []zapcore.Field) {
	m := make(map[string]string, 0)
	m["msg"] = msg
	for _, element := range fields {
		m[element.Key] = element.String
	}
	job.WarnLog = append(job.WarnLog, m)
}

func (job *MigrationJob) initTenant(tenant Tenant) {
	mAudit := &MigrationAudit{
		MigratedEntities:    make([]MigrationAuditEntry, 0),
		NonMigratedEntities: make([]NonMigrationAuditEntry, 0),
	}
	job.Audit[tenant.Url] = mAudit
}

func (job *MigrationJob) registerNonMigratedSoftwareList(tenant Tenant, e []SoftwareMigration, msg string) {
	for _, i := range e {
		job.registerNonMigratedSoftware(tenant, i, msg)
	}
}

func (job *MigrationJob) registerNonMigratedSoftware(tenant Tenant, e SoftwareMigration, msg string) {
	job.Audit[tenant.Url].NonMigratedEntities = append(job.Audit[tenant.Url].NonMigratedEntities, NonMigrationAuditEntry{
		SoftwareName:    e.Software.Name,
		SoftwareVersion: e.Software.Version,
		SoftwareUrl:     e.Software.Url,
		Reason:          msg,
	})
}

func (job *MigrationJob) registerMigratedSoftware(tenant Tenant, e SoftwareMigration, createdSwId string, createdSwVersionId string) {
	job.Audit[tenant.Url].MigratedEntities = append(job.Audit[tenant.Url].MigratedEntities, MigrationAuditEntry{
		SoftwareName:           e.Software.Name,
		SoftwareVersion:        e.Software.Version,
		OldSoftwareMoId:        e.Software.Id,
		NewSoftwareMoId:        createdSwId,
		NewSoftwareVersionMoId: createdSwVersionId,
	})
}

func (job *MigrationJob) isConnectionProbeSucceeded(tenant Tenant, b bool) {
	job.Audit[tenant.Url].ConnectionProbeSucceeded = b
}

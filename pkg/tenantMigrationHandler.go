package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/reubenmiller/go-c8y-cli-microservice/pkg/c8ycli"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ByCreationTime []SoftwareMigration

func (a ByCreationTime) Len() int { return len(a) }
func (a ByCreationTime) Less(i, j int) bool {
	return a[i].Software.CreationTime.Before(a[j].Software.CreationTime)
}
func (a ByCreationTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type AbortionReason struct {
	err         error
	errorFields []zapcore.Field
}

func (r *AbortionReason) Error() string {
	return fmt.Sprintf("err %v", r.err)
}

func isOldSoftwarePackage(e map[string]interface{}) bool {
	childAdditions := e["childAdditions"]
	if childAdditions == nil {
		return true
	}
	cChildAdditions, ok := childAdditions.(map[string]interface{})
	if !ok {
		return true
	}

	refs := cChildAdditions["references"]
	if refs == nil {
		return true
	}
	cRefs, ok := refs.([]interface{})
	if !ok {
		return true
	}

	if len(cRefs) == 0 {
		return true
	}

	return false
}

func migrateSoftwareGroup(executor *c8ycli.Executor, job *MigrationJob, tenant Tenant, swName string, swMigrations *[]SoftwareMigration) {
	Log("Migration of Software Repository Group started", INFO, job, tenant, []zapcore.Field{zap.String("swName", swName), zap.Int("countVersion", len(*swMigrations))})

	executor.Command = fmt.Sprintf("c8y inventory create --template ./templates/template_software_parent_new.jsonnet --templateVars 'name=%s' --select id -o csv -f --withError", swName)
	result, err := executor.Execute(true)
	if err != nil || result.ExitCode != 0 {
		Log("Error while creating new software package. Skipping Software Group "+swName, ERROR, job, tenant, ExtractZapLogs(executor, result))
		return
	}
	newSwId := strings.TrimRight(string(result.Stdout), "\n") //byteToMaps(result.Stdout)[0]["id"].(string)
	Log("Created software package", INFO, job, tenant, []zapcore.Field{zap.String("newSwId", newSwId), zap.String("tenant", tenant.Url)})

	sort.Sort(ByCreationTime(*swMigrations))

	for _, e := range *swMigrations {
		if aborted, mErr := migrateSoftwareVersion(executor, job, &e, newSwId, tenant); aborted {
			Log(mErr.Error(), ERROR, job, tenant, mErr.errorFields)
			Log("Aborted Migration of Software Version", INFO, job, tenant, []zapcore.Field{zap.Any("SwMigration", e)})
		}
	}
	Log("Migration of Software Repository Group finished", INFO, job, tenant, []zapcore.Field{zap.String("swName", swName), zap.Int("countVersion", len(*swMigrations))})
}

func migrateSoftwareVersion(executor *c8ycli.Executor, job *MigrationJob, swMigration *SoftwareMigration, newSwId string, tenant Tenant) (aborted bool, ame AbortionReason) {
	newSwVersionId, aborted, mErr := createSoftwareVersion(executor, swMigration, job, tenant)
	if aborted {
		return true, mErr
	} else if len(newSwVersionId) == 0 {
		return true, AbortionReason{errors.New("Error while created Software Version Managed Object for Software MO ID = " + newSwId + ". newSwVersionId.length = 0"), []zapcore.Field{}}
	}

	assignSoftwareVersionToSoftware(executor, newSwId, newSwVersionId, job, tenant)

	logText := fmt.Sprintf("Software Object was deprecated and replaced by a new one (ID:%s)", newSwId)
	burrySoftwareObject(executor, swMigration, logText, job, tenant)
	createAuditLogEntry(executor, logText, swMigration, job, tenant)

	job.registerMigratedSoftware(tenant, *swMigration, newSwId, newSwVersionId)
	swMigration.Migrated = true
	Log("Migrated Software Repository entry", INFO, job, tenant, []zapcore.Field{zap.String("oldSwId", swMigration.Software.Id), zap.String("swName", swMigration.Software.Name),
		zap.String("swVersion", swMigration.Software.Version), zap.String("newSoftwareId", newSwId), zap.String("newSwVersionId", newSwVersionId)})

	return false, AbortionReason{}
}

func createSoftwareVersion(executor *c8ycli.Executor, swMigration *SoftwareMigration, job *MigrationJob, tenant Tenant) (swvId string, aborted bool, ame AbortionReason) {
	executor.Command = fmt.Sprintf("c8y inventory create --template ./templates/template_software_version_new.jsonnet --templateVars 'url=%s,version=%s' --select id -o csv -f --withError",
		swMigration.Software.Url, swMigration.Software.Version)
	result, err := executor.Execute(true)
	if err != nil || result.ExitCode != 0 {
		errText := "Error while updating software object " + swMigration.Software.Name + ". Aborting this version."
		return "", true, AbortionReason{errors.New(errText), ExtractZapLogs(executor, result)}
	}
	newSwVersionId := strings.TrimRight(string(result.Stdout), "\n")
	Log("Created software version managedObject", INFO, job, tenant, []zapcore.Field{zap.String("swVersionId", newSwVersionId)})
	return newSwVersionId, false, AbortionReason{}
}

func assignSoftwareVersionToSoftware(executor *c8ycli.Executor, newSwId string, newSwVersionId string, job *MigrationJob, tenant Tenant) {
	executor.Command = fmt.Sprintf("c8y inventory additions assign --id %s --newChild %s -f --withError", newSwId, newSwVersionId)
	result, err := executor.Execute(true)
	if err != nil || result.ExitCode != 0 {
		Log("Error while assigining sw version as addition", ERROR, job, tenant, ExtractZapLogs(executor, result))
	}
	Log("Assigned software version to software package", INFO, job, tenant, []zapcore.Field{zap.String("swVersionId", newSwVersionId), zap.String("swMoId", newSwId)})
}

func burrySoftwareObject(executor *c8ycli.Executor, swMigration *SoftwareMigration, logText string, job *MigrationJob, tenant Tenant) {
	executor.Command = fmt.Sprintf("c8y inventory update --id %s --data 'type=c8y_SoftwareDeprecated,note=%s,status=BURRIED' -f --withError", swMigration.Software.Id, logText)
	result, err := executor.Execute(true)
	if err != nil || result.ExitCode != 0 {
		Log("Error while burrying old software object", ERROR, job, tenant, ExtractZapLogs(executor, result))
	}
	Log("Burried old software package", INFO, job, tenant, []zapcore.Field{zap.String("burriedMoId", swMigration.Software.Id)})
}

func createAuditLogEntry(executor *c8ycli.Executor, msg string, swMigration *SoftwareMigration, job *MigrationJob, tenant Tenant) {
	executor.Command = fmt.Sprintf("c8y auditrecords create --type Inventory --text '%s' --source %s --activity 'Managed Object updated' --severity 'information' -f --withError",
		msg, swMigration.Software.Id)
	result, err := executor.Execute(true)
	if err != nil || result.ExitCode != 0 {
		Log("Error while creating audit log entry", ERROR, job, tenant, ExtractZapLogs(executor, result))
	}
	Log("Created audit log entry for burry action", INFO, job, tenant, []zapcore.Field{zap.String("burriedMoId", swMigration.Software.Id)})
}

func probeConnection(e *c8ycli.Executor) (bool, *c8ycli.ExecutorResult, error) {
	e.Command = "c8y inventory list --withError"
	result, err := e.Execute(true)
	if err != nil {
		return false, result, err
	}
	if result.ExitCode != 0 {
		return false, result, fmt.Errorf("exit code = %d; stderr = %s", result.ExitCode, string(result.Stderr))
	}
	stdout := string(result.Stdout)
	if len(stdout) == 0 {
		return false, result, errors.New("no stdout output for probe command (" + e.Command + ")")
	}
	return true, result, nil
}

func newExecutor(tenant Tenant) *c8ycli.Executor {
	return &c8ycli.Executor{
		Options: &c8ycli.CLIOptions{
			Host:         tenant.Url,
			Username:     tenant.User,
			Password:     tenant.Pass,
			EnableCreate: true,
			EnableUpdate: true,
			EnableDelete: false,
		},
	}
}

func migrateTenant(tenant Tenant, job *MigrationJob) {
	Log("Tenant migration started", INFO, job, tenant, []zapcore.Field{})

	job.initTenant(tenant)

	// Setup executor to given tenant
	executor := newExecutor(tenant)
	Log(fmt.Sprintf("Set executor settings to tenant: %+v", *executor.Options), INFO, job, tenant, []zapcore.Field{})

	// probe tenant connection
	Log("Probe tenant connection ...", INFO, job, tenant, []zapcore.Field{})
	c, result, _ := probeConnection(executor)
	if !c {
		Log("Connection probe for tenant failed -> cancelling tenant migration", ERROR, job, tenant, ExtractZapLogs(executor, result))
		return
	}
	job.isConnectionProbeSucceeded(tenant, true)
	Log("Connection probe succeeded", INFO, job, tenant, []zapcore.Field{})

	// fetch and categorize softwares
	executor.Command = "c8y inventory list --type c8y_Software --includeAll --withError"
	result, err := executor.Execute(true)
	if err != nil || result.ExitCode != 0 {
		Log("Error while fetching c8y Software Objects. Skipping tenant "+tenant.Url, ERROR, job, tenant, ExtractZapLogs(executor, result))
	}
	swEntrys := byteToMaps(result.Stdout)
	if len(swEntrys) == 0 {
		Log("Could not find any Software object. Tenant migration skipped.", WARN, job, tenant, ExtractZapLogs(executor, result))
	}
	swMigrationElements := Convert(swEntrys)
	migrationRelevant, nonMigrationRelevant := Filter(swMigrationElements, func(sm SoftwareMigration) bool {
		return sm.IsOldPackage
	})
	job.registerNonMigratedSoftwareList(tenant, nonMigrationRelevant, "Classified as non-migration relevant")

	// Group (by Name) and migrate software
	for swName, swMigrations := range GroupByName(migrationRelevant) {
		migrateSoftwareGroup(executor, job, tenant, swName, &swMigrations)
	}

	Log("Tenant migration finished", INFO, job, tenant, []zapcore.Field{})
}

func BulkMigration(job *MigrationJob) {
	job.Status = "RUNNING"

	maxPoolSize := 2
	var wg sync.WaitGroup

	jobs := make(chan Tenant)
	for i := 0; i < min(maxPoolSize, len(job.Tenants)); i++ {
		go func(id int, wg *sync.WaitGroup, in <-chan Tenant) {
			for j := range in {
				migrateTenant(j, job)
				wg.Done()
			}
		}(i, &wg, jobs)
	}
	for _, tenant := range job.Tenants {
		wg.Add(1)
		jobs <- tenant
	}
	close(jobs)
	wg.Wait()

	if len(job.ErrorLog) == 0 {
		job.Status = "COMPLETED"
	} else {
		job.Status = "COMPLETED_WITH_ERRORS"
	}

}

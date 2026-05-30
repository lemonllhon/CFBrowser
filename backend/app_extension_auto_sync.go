package backend

import (
	"strings"
	"time"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/logger"
)

func (a *App) scheduleAutoSyncSharedExtensionDataAfterProfileStopped(profileId string) {
	if a == nil {
		return
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.autoSyncSharedExtensionDataAfterProfileStopped(profileId)
	}()
}

func (a *App) autoSyncSharedExtensionDataAfterProfileStopped(profileId string) {
	profileId = strings.TrimSpace(profileId)
	if a == nil || profileId == "" {
		return
	}
	dao, err := a.extensionDAO()
	if err != nil {
		logger.New("Extension").Error("自动数据同步初始化失败", logger.F("profile_id", profileId), logger.F("error", err.Error()))
		return
	}
	sourceBindings, err := dao.ListBindingsByProfile(profileId)
	if err != nil {
		logger.New("Extension").Error("自动数据同步读取绑定失败", logger.F("profile_id", profileId), logger.F("error", err.Error()))
		return
	}
	for _, sourceBinding := range sourceBindings {
		if !isEnabledSharedExtensionBinding(sourceBinding) {
			continue
		}
		a.autoSyncSharedExtensionBindingData(dao, sourceBinding)
	}
}

func (a *App) autoSyncSharedExtensionBindingData(dao browser.ExtensionDAO, sourceBinding browser.ExtensionBinding) {
	log := logger.New("Extension")
	allBindings, err := dao.ListBindings(sourceBinding.ExtensionId)
	if err != nil {
		log.Error("自动数据同步读取扩展绑定失败",
			logger.F("extension_id", sourceBinding.ExtensionId),
			logger.F("profile_id", sourceBinding.ProfileId),
			logger.F("error", err.Error()),
		)
		return
	}
	if a.isExtensionAutoSyncBlocked(sourceBinding.ExtensionId) {
		log.Info("跳过自动数据同步：共享扩展已等待手动同步",
			logger.F("extension_id", sourceBinding.ExtensionId),
			logger.F("source_profile_id", sourceBinding.ProfileId),
		)
		return
	}

	runningSharedProfiles := make([]string, 0)
	targetProfileIds := make([]string, 0)
	for _, binding := range allBindings {
		if !isEnabledSharedExtensionBinding(binding) || strings.EqualFold(binding.ProfileId, sourceBinding.ProfileId) {
			continue
		}
		if a.isProfileRunning(binding.ProfileId) {
			runningSharedProfiles = append(runningSharedProfiles, binding.ProfileId)
			continue
		}
		targetProfileIds = append(targetProfileIds, binding.ProfileId)
	}

	if len(runningSharedProfiles) > 0 {
		a.markExtensionAutoSyncBlocked(sourceBinding.ExtensionId)
		log.Info("跳过自动数据同步：存在其他运行中的共享实例",
			logger.F("extension_id", sourceBinding.ExtensionId),
			logger.F("source_profile_id", sourceBinding.ProfileId),
			logger.F("running_profile_ids", strings.Join(runningSharedProfiles, ",")),
		)
		return
	}
	if len(targetProfileIds) == 0 {
		return
	}

	if _, err := a.BrowserExtensionSyncProfileData(BrowserExtensionSyncDataInput{
		ExtensionId:      sourceBinding.ExtensionId,
		SourceProfileId:  sourceBinding.ProfileId,
		TargetProfileIds: targetProfileIds,
	}); err != nil {
		log.Error("自动数据同步失败",
			logger.F("extension_id", sourceBinding.ExtensionId),
			logger.F("source_profile_id", sourceBinding.ProfileId),
			logger.F("target_profile_ids", strings.Join(targetProfileIds, ",")),
			logger.F("error", err.Error()),
		)
		return
	}

	log.Info("自动数据同步完成",
		logger.F("extension_id", sourceBinding.ExtensionId),
		logger.F("source_profile_id", sourceBinding.ProfileId),
		logger.F("target_profile_ids", strings.Join(targetProfileIds, ",")),
	)
	if a.ctx != nil {
		a.emitEvent("browser:extension:auto-data-synced", map[string]interface{}{
			"extensionId":      sourceBinding.ExtensionId,
			"sourceProfileId":  sourceBinding.ProfileId,
			"targetProfileIds": targetProfileIds,
			"targetCount":      len(targetProfileIds),
		})
	}
}

func isEnabledSharedExtensionBinding(binding browser.ExtensionBinding) bool {
	return binding.Enabled && normalizeExtensionBindingMode(binding.Mode) == "shared"
}

func (a *App) markExtensionAutoSyncBlocked(extensionID string) {
	extensionID = strings.TrimSpace(extensionID)
	if a == nil || extensionID == "" {
		return
	}
	a.extensionAutoSyncMu.Lock()
	if a.extensionAutoSyncBlocked == nil {
		a.extensionAutoSyncBlocked = make(map[string]struct{})
	}
	a.extensionAutoSyncBlocked[extensionID] = struct{}{}
	a.extensionAutoSyncMu.Unlock()
}

func (a *App) clearExtensionAutoSyncBlocked(extensionID string) {
	extensionID = strings.TrimSpace(extensionID)
	if a == nil || extensionID == "" {
		return
	}
	a.extensionAutoSyncMu.Lock()
	delete(a.extensionAutoSyncBlocked, extensionID)
	a.extensionAutoSyncMu.Unlock()
}

func (a *App) isExtensionAutoSyncBlocked(extensionID string) bool {
	extensionID = strings.TrimSpace(extensionID)
	if a == nil || extensionID == "" {
		return false
	}
	a.extensionAutoSyncMu.Lock()
	_, blocked := a.extensionAutoSyncBlocked[extensionID]
	a.extensionAutoSyncMu.Unlock()
	return blocked
}

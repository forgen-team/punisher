import { checkErrorMessages, settingsErrorMessages } from "../i18n";
import { liveLabels } from "./liveLabels";

/** 확인·복습 API 오류 code → 안내(describeCheckError) */
export type CheckErrorCode =
  | "ANSWER_EMPTY"
  | "ANSWER_TOO_LONG"
  | "REASON_TOO_LONG"
  | "INVALID_VERDICT"
  | "NO_OBJECTIVES"
  | "INVALID_KIND"
  | "NO_REVIEWS"
  | "NO_PLACEMENT"
  | "MISSING_TARGET"
  | "CHOICE_REQUIRED"
  | "INVALID_CHOICE"
  | "TURN_REQUIRED"
  | "INVALID_STATUS"
  | "INVALID_CONFIDENCE"
  | "NO_SYNTHESIS"
  | "BAD_REQUEST";

/** 현재 언어를 따른다(문구 카탈로그: ../i18n checkErrorMessages). */
export const CHECK_ERROR_MESSAGES: Record<CheckErrorCode, string> = liveLabels(checkErrorMessages);

/** 설정 > 학습 PUT 400 code → 안내(describeSettingsError) */
export type SettingsErrorCode =
  | "INVALID_LANGUAGE"
  | "INVALID_GRADER"
  | "INVALID_DAILY_MINUTES"
  | "INVALID_MASTERY_TARGET"
  | "INVALID_INTERVALS"
  | "INVALID_REVIEW_MINUTES"
  | "INVALID_LAB_OS"
  | "INVALID_READER_LEVEL"
  | "INVALID_AREA_FAMILIARITY"
  | "INVALID_AI_PROVIDER"
  | "SETTINGS_REQUIRED"
  | "BAD_REQUEST";

/** 현재 언어를 따른다(문구 카탈로그: ../i18n settingsErrorMessages). */
export const SETTINGS_ERROR_MESSAGES: Record<SettingsErrorCode, string> = liveLabels(settingsErrorMessages);

export type Language = "node" | "php" | "python" | "bun" | "go" | "dotnet" | "deno" | "git" | "rust" | "composer" | "yarn" | "pnpm" | "vcpkg" | "chocolatey";
export type Runtime = {
  language: Language;
  installed: string[];
  project_version: string;
  global_version: string;
};
export type Diagnosis = {
  language: Language;
  version?: string;
  source: string;
  config_path?: string;
  installed: boolean;
  expected_executable?: string;
  path_executable?: string;
  path_matches: boolean;
  uses_shim?: boolean;
  status: string;
  detail?: string;
};
export type Snapshot = {
  directory: string;
  config_path: string;
  tools: Runtime[];
  diagnosis: {
    issues: boolean;
    tools: Diagnosis[];
    warnings: string[];
    virtual_environment?: string;
  };
};
export type Release = {
  version: string;
  build: string;
  variant: string;
  installed: boolean;
  policy: {
    provider: string;
    trust: string;
    installable: boolean;
    unavailable_reason?: string;
    support: string;
    eol?: string;
    policy_source: string;
    policy_reviewed: string;
  };
};
export type Catalog = {
  language: Language;
  versions: Release[];
  fetched_at: string;
  warning: string;
  from_cache: boolean;
  stale: boolean;
};
export type Request = {
  action: string;
  language: Language;
  version: string;
  directory: string;
  scope: "project" | "global";
  allow_selected: boolean;
};
export type Operation = {
  id: string;
  request: Request;
  status: "running" | "succeeded" | "failed" | "canceled";
  phase: string;
  bytes: number;
  total: number;
  message: string;
  error_code?: string;
  hint?: string;
  started: string;
  finished?: string;
};
export type Setup = {
  version: string;
  cli_path: string;
  cli_exists: boolean;
  shims: Record<Language, boolean>;
  message: string;
};
export type Bridge = {
  Snapshot(dir: string): Promise<Snapshot>;
  Catalog(
    lang: Language,
    history: boolean,
    refresh: boolean,
    offline: boolean,
  ): Promise<Catalog>;
  Start(r: Request): Promise<string>;
  Operations(): Promise<Operation[]>;
  Cancel(id: string): Promise<void>;
  PickProject(): Promise<string>;
  SetupInfo(): Promise<Setup>;
  Documentation(topic: string): Promise<string>;
  OpenTerminal(dir: string): Promise<void>;
};
declare global {
  interface Window {
    go?: { main: { App: Bridge } };
  }
}

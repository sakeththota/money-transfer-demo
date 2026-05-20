const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:7070';

export interface ServerInfo {
  address: string;
  namespace: string;
  taskQueue: string;
  encryptPayloads: boolean;
  secureConnection: boolean;
  codecServerUrl?: string;
}

export interface TransferParams {
  amount: number;
  fromAccount: string;
  toAccount: string;
  scenario: string;
}

export interface TransferStatus {
  progressPercentage: number;
  transferState: string;
  workflowStatus: string;
  chargeResult?: {
    chargeId: string;
  };
  approvalTime?: number;
}

export interface WorkflowStatus {
  workflowId: string;
  status: string;
}

export interface ScheduleParams extends TransferParams {
  intervalHours: number;
}

export async function getServerInfo(): Promise<ServerInfo> {
  const res = await fetch(`${API_BASE}/serverinfo`);
  if (!res.ok) throw new Error('Failed to fetch server info');
  return res.json();
}

export async function runWorkflow(params: TransferParams): Promise<{ transferId: string }> {
  const res = await fetch(`${API_BASE}/runWorkflow`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  });
  if (!res.ok) throw new Error('Failed to start workflow');
  return res.json();
}

export async function queryWorkflow(workflowId: string): Promise<TransferStatus> {
  const res = await fetch(`${API_BASE}/runQuery`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workflowId }),
  });
  if (!res.ok) throw new Error('Failed to query workflow');
  return res.json();
}

export async function approveTransfer(workflowId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/approveTransfer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workflowId }),
  });
  if (!res.ok) throw new Error('Failed to approve transfer');
}

export async function listWorkflows(): Promise<WorkflowStatus[]> {
  const res = await fetch(`${API_BASE}/listWorkflows`);
  if (!res.ok) throw new Error('Failed to list workflows');
  return res.json();
}

export async function scheduleWorkflow(params: ScheduleParams): Promise<{ transferId: string }> {
  const res = await fetch(`${API_BASE}/scheduleWorkflow`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  });
  if (!res.ok) throw new Error('Failed to schedule workflow');
  return res.json();
}

export interface ScheduleInfo {
  scheduleId: string;
  nextRunTime: string | null;
  paused: boolean;
}

export async function getScheduleInfo(scheduleId: string): Promise<ScheduleInfo> {
  const res = await fetch(`${API_BASE}/scheduleInfo/${scheduleId}`);
  if (!res.ok) throw new Error('Failed to get schedule info');
  return res.json();
}

export interface ScheduleStatus {
  scheduleId: string;
  nextRunTime: string | null;
  paused: boolean;
}

export async function listSchedules(): Promise<ScheduleStatus[]> {
  const res = await fetch(`${API_BASE}/listSchedules`);
  if (!res.ok) throw new Error('Failed to list schedules');
  return res.json();
}

export async function deleteSchedule(scheduleId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/schedule/${scheduleId}`, {
    method: 'DELETE',
  });
  if (!res.ok) throw new Error('Failed to delete schedule');
}

export async function getBalances(): Promise<Record<string, number>> {
  const res = await fetch(`${API_BASE}/balances`);
  if (!res.ok) throw new Error('Failed to fetch balances');
  return res.json();
}

export async function resetBalances(): Promise<void> {
  const res = await fetch(`${API_BASE}/resetBalances`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to reset balances');
}

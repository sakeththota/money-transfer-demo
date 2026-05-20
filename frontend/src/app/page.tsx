"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import {
  getServerInfo,
  runWorkflow,
  queryWorkflow,
  approveTransfer,
  listWorkflows,
  listSchedules,
  scheduleWorkflow,
  getScheduleInfo,
  deleteSchedule,
  getBalances,
  resetBalances,
  ServerInfo,
  TransferStatus,
  WorkflowStatus,
  ScheduleInfo,
  ScheduleStatus,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  ArrowRight,
  Building2,
  CheckCircle2,
  Clock,
  CreditCard,
  ExternalLink,
  Loader2,
  RefreshCw,
  RotateCcw,
  Send,
  ShieldCheck,
  Trash2,
  Wallet,
  XCircle,
  AlertTriangle,
} from "lucide-react";

const SCENARIOS = [
  { value: "HAPPY_PATH", label: "Happy Path", description: "Normal execution" },
  { value: "ADVANCED_VISIBILITY", label: "Advanced Visibility", description: "Search attributes" },
  { value: "HUMAN_IN_LOOP", label: "Human-In-Loop", description: "Requires approval" },
  { value: "API_DOWNTIME", label: "API Downtime", description: "Retries on failure" },
  { value: "BUG_IN_WORKFLOW", label: "Bug in Workflow", description: "Recoverable error" },
  { value: "SAGA_ROLLBACK", label: "Saga Pattern", description: "Compensation flow" },
];

const SENDER_NAME = process.env.NEXT_PUBLIC_SENDER_NAME || "User";

const FROM_ACCOUNTS = [
  { id: "checking", name: "Checking Account", number: "****6789" },
  { id: "savings", name: "Savings Account", number: "****5566" },
];

const TO_ACCOUNTS = [
  { id: "justine", name: "Justine Morris", initials: "JM", number: "****7654" },
  { id: "raul", name: "Raul Ruidiaz", initials: "RR", number: "****9988" },
  { id: "ian", name: "Ian Wu", initials: "IW", number: "****3456" },
  { id: "emma", name: "Emma Stockton", initials: "ES", number: "****2233" },
];

// Countdown Timer Component
function CountdownTimer({ targetTime, label, onExpire }: {
  targetTime: Date;
  label?: string;
  onExpire?: () => void;
}) {
  const [timeLeft, setTimeLeft] = useState<number>(0);

  useEffect(() => {
    const calculateTimeLeft = () => {
      const diff = targetTime.getTime() - Date.now();
      return Math.max(0, Math.floor(diff / 1000));
    };

    setTimeLeft(calculateTimeLeft());
    const interval = setInterval(() => {
      const newTimeLeft = calculateTimeLeft();
      setTimeLeft(newTimeLeft);
      if (newTimeLeft === 0 && onExpire) {
        onExpire();
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [targetTime, onExpire]);

  const formatTime = (seconds: number) => {
    if (seconds >= 3600) {
      const h = Math.floor(seconds / 3600);
      const m = Math.floor((seconds % 3600) / 60);
      const s = seconds % 60;
      return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
    }
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  return (
    <div className="flex items-center gap-2 font-mono text-sm">
      <Clock className="h-4 w-4 text-muted-foreground" />
      {label && <span className="text-muted-foreground">{label}</span>}
      <span className={timeLeft <= 10 ? "text-amber-600 font-semibold" : ""}>
        {formatTime(timeLeft)}
      </span>
    </div>
  );
}

const truncateServerAddress = (address: string | undefined) => {
  if (!address) return "...";
  // For cloud addresses like "namespace.accountId.tmprl.cloud:7233"
  // Show just the namespace part with "..."
  const parts = address.split(".");
  if (parts.length >= 2 && address.includes("tmprl.cloud")) {
    return `${parts[0]}...`;
  }
  return address;
};

const getWorkflowUrl = (workflowId: string, serverInfo: ServerInfo | null) => {
  if (!serverInfo) return "#";
  const { namespace, secureConnection } = serverInfo;
  const isSchedule = workflowId.startsWith("schedule-");
  const resourceType = isSchedule ? "schedules" : "workflows";
  if (secureConnection) {
    return `https://cloud.temporal.io/namespaces/${namespace}/${resourceType}/${workflowId}`;
  }
  return `http://localhost:8233/namespaces/${namespace}/${resourceType}/${workflowId}`;
};

export default function Home() {
  const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null);
  const [fromAccount, setFromAccount] = useState(FROM_ACCOUNTS[0].name);
  const [toAccount, setToAccount] = useState(TO_ACCOUNTS[0].name);
  const [amount, setAmount] = useState("100");
  const [scenario, setScenario] = useState(SCENARIOS[0].value);
  const [isSchedule, setIsSchedule] = useState(false);
  const [intervalHours, setIntervalHours] = useState("24");

  const [activeWorkflowId, setActiveWorkflowId] = useState<string | null>(null);
  const [transferStatus, setTransferStatus] = useState<TransferStatus | null>(null);
  const [workflows, setWorkflows] = useState<WorkflowStatus[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [scheduleInfo, setScheduleInfo] = useState<ScheduleInfo | null>(null);
  const [approvalStartTime, setApprovalStartTime] = useState<Date | null>(null);
  const [approvalSent, setApprovalSent] = useState(false);
  const [schedules, setSchedules] = useState<ScheduleStatus[]>([]);
  const [balances, setBalances] = useState<Record<string, number>>({});
  const [flashState, setFlashState] = useState<Record<string, "credit" | "debit">>({});
  const prevBalancesRef = useRef<Record<string, number>>({});

  const refreshBalances = useCallback(() => {
    getBalances()
      .then((newBalances) => {
        const prev = prevBalancesRef.current;
        const flashes: Record<string, "credit" | "debit"> = {};

        for (const [name, balance] of Object.entries(newBalances)) {
          if (prev[name] !== undefined && balance !== prev[name]) {
            flashes[name] = balance > prev[name] ? "credit" : "debit";
          }
        }

        prevBalancesRef.current = newBalances;
        setBalances(newBalances);

        if (Object.keys(flashes).length > 0) {
          setFlashState(flashes);
          setTimeout(() => setFlashState({}), 700);
        }
      })
      .catch((err) => console.error("Failed to fetch balances:", err));
  }, []);

  useEffect(() => {
    getServerInfo()
      .then(setServerInfo)
      .catch((err) => console.error("Failed to get server info:", err));
    refreshBalances();
  }, [refreshBalances]);

  const refreshWorkflows = useCallback(() => {
    listWorkflows()
      .then((data) => setWorkflows(data || []))
      .catch((err) => console.error("Failed to list workflows:", err));
    listSchedules()
      .then((data) => setSchedules(data || []))
      .catch((err) => console.error("Failed to list schedules:", err));
  }, []);

  useEffect(() => {
    refreshWorkflows();
    const interval = setInterval(refreshWorkflows, 5000);
    return () => clearInterval(interval);
  }, [refreshWorkflows]);

  // Poll balances: 1s when a workflow is active, 5s when idle
  useEffect(() => {
    const interval = setInterval(refreshBalances, activeWorkflowId ? 1000 : 5000);
    return () => clearInterval(interval);
  }, [activeWorkflowId, refreshBalances]);

  // Fetch schedule info when active workflow is a schedule
  useEffect(() => {
    // Clear previous schedule info immediately when ID changes
    setScheduleInfo(null);

    if (!activeWorkflowId || !activeWorkflowId.startsWith("schedule-")) {
      return;
    }

    let isActive = true;
    const fetchScheduleInfo = async () => {
      try {
        const info = await getScheduleInfo(activeWorkflowId);
        if (isActive) setScheduleInfo(info);
      } catch (err) {
        console.error("Failed to get schedule info:", err);
        if (isActive) setScheduleInfo(null);
      }
    };

    fetchScheduleInfo();
    const interval = setInterval(fetchScheduleInfo, 5000);
    return () => {
      isActive = false;
      clearInterval(interval);
    };
  }, [activeWorkflowId]);

  // Track when approval state starts for countdown
  useEffect(() => {
    if (transferStatus?.transferState === "waiting" && !approvalStartTime) {
      setApprovalStartTime(new Date());
    } else if (transferStatus?.transferState !== "waiting") {
      setApprovalStartTime(null);
    }
  }, [transferStatus?.transferState, approvalStartTime]);

  // Reset approvalSent when workflow leaves "waiting" state
  useEffect(() => {
    if (transferStatus?.transferState !== "waiting") {
      setApprovalSent(false);
    }
  }, [transferStatus?.transferState]);

  useEffect(() => {
    if (!activeWorkflowId) return;

    // Clear previous status when switching workflows
    setTransferStatus(null);
    setApprovalStartTime(null);
    setApprovalSent(false);

    // Track if this effect is still active (prevents stale updates)
    let isActive = true;
    let intervalId: ReturnType<typeof setInterval> | null = null;

    const isTerminal = (status: TransferStatus) =>
      status.transferState === "finished" ||
      status.transferState === "compensated" ||
      status.workflowStatus === "FAILED" ||
      status.workflowStatus === "COMPLETED";

    const pollStatus = async () => {
      try {
        const status = await queryWorkflow(activeWorkflowId);
        if (!isActive) return;

        setTransferStatus(status);
        if (isTerminal(status)) {
          refreshWorkflows();
          refreshBalances();
          setApprovalStartTime(null);
          // Stop polling once workflow is done
          if (intervalId) {
            clearInterval(intervalId);
            intervalId = null;
          }
        }
      } catch (err) {
        console.error("Failed to query workflow:", err);
        if (isActive) {
          setApprovalStartTime(null);
          refreshWorkflows();
        }
      }
    };

    pollStatus();
    intervalId = setInterval(pollStatus, 1000);

    return () => {
      isActive = false;
      if (intervalId) clearInterval(intervalId);
    };
  }, [activeWorkflowId, refreshWorkflows, refreshBalances]);

  const handleTransfer = async () => {
    setIsLoading(true);
    setError(null);
    setTransferStatus(null);

    try {
      const params = {
        amount: Number(amount),
        fromAccount,
        toAccount,
        scenario,
      };
      const result = isSchedule
        ? await scheduleWorkflow({ ...params, intervalHours: Number(intervalHours) })
        : await runWorkflow(params);
      setActiveWorkflowId(result.transferId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Transfer failed");
    } finally {
      setIsLoading(false);
    }
  };

  const handleApprove = async () => {
    if (!activeWorkflowId) return;
    try {
      setApprovalSent(true); // Immediately hide approval UI
      await approveTransfer(activeWorkflowId);
    } catch (err) {
      setApprovalSent(false); // Show approval UI again if it failed
      setError(err instanceof Error ? err.message : "Approval failed");
    }
  };

  const handleResetBalances = async () => {
    try {
      await resetBalances();
      refreshBalances();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to reset balances");
    }
  };

  const handleDeleteSchedule = async (scheduleId: string, e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent selecting the schedule
    try {
      await deleteSchedule(scheduleId);
      if (activeWorkflowId === scheduleId) {
        setActiveWorkflowId(null);
        setScheduleInfo(null);
      }
      refreshWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete schedule");
    }
  };

  const needsApproval = !approvalSent &&
    transferStatus?.transferState === "waiting" &&
    !transferStatus?.workflowStatus?.includes("FAILED") &&
    !transferStatus?.workflowStatus?.includes("COMPLETED");
  const isLargeTransfer = Number(amount) > 1000;
  const selectedFromAccount = FROM_ACCOUNTS.find((a) => a.name === fromAccount);
  const selectedToAccount = TO_ACCOUNTS.find((a) => a.name === toAccount);

  const getStatusDisplay = () => {
    if (!transferStatus) return null;

    // Completed states
    if (transferStatus.transferState === "finished") {
      return { label: "Completed", variant: "success" as const, icon: CheckCircle2 };
    }
    if (transferStatus.workflowStatus === "COMPLETED") {
      return { label: "Completed", variant: "success" as const, icon: CheckCircle2 };
    }
    // Saga compensation states - workflow still fails even after compensation
    if (transferStatus.transferState === "compensated" || transferStatus.transferState === "compensating") {
      return { label: "Failed", variant: "destructive" as const, icon: XCircle };
    }
    // Rejected (large transfer not approved in time)
    if (transferStatus.transferState === "rejected") {
      return { label: "Rejected", variant: "destructive" as const, icon: XCircle };
    }
    // Failed state
    if (transferStatus.workflowStatus === "FAILED") {
      return { label: "Failed", variant: "destructive" as const, icon: XCircle };
    }
    // Waiting for approval (but not if we already sent approval)
    if (transferStatus.transferState === "waiting" && !approvalSent) {
      return { label: "Awaiting Approval", variant: "warning" as const, icon: Clock };
    }
    // Default: still processing
    return { label: "Processing", variant: "info" as const, icon: RefreshCw };
  };

  const statusDisplay = getStatusDisplay();

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted/30">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4 md:px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary">
              <Building2 className="h-5 w-5 text-primary-foreground" />
            </div>
            <div className="hidden sm:block">
              <h1 className="text-lg font-semibold">Temporal Bank</h1>
              <p className="text-xs text-muted-foreground">Secure Money Transfers</p>
            </div>
          </div>

          <Badge variant="outline" className="gap-1.5">
            <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
            <span className="hidden sm:inline">{serverInfo?.namespace || "Connecting..."}</span>
          </Badge>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-6 md:px-6 md:py-8">
        {/* Account Balance Cards */}
        <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <Card className={`transition-colors ${flashState[fromAccount] === "debit" ? "animate-flash-debit" : flashState[fromAccount] === "credit" ? "animate-flash-credit" : ""}`}>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">From: {fromAccount}</p>
                  <p className={`text-2xl font-bold tabular-nums transition-colors ${flashState[fromAccount] === "debit" ? "text-red-600" : flashState[fromAccount] === "credit" ? "text-emerald-600" : ""}`}>
                    {balances[fromAccount] !== undefined
                      ? `$${balances[fromAccount].toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
                      : "..."}
                  </p>
                </div>
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                  <Wallet className="h-5 w-5 text-primary" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card className={`transition-colors ${flashState[toAccount] === "credit" ? "animate-flash-credit" : flashState[toAccount] === "debit" ? "animate-flash-debit" : ""}`}>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">To: {toAccount}</p>
                  <p className={`text-2xl font-bold tabular-nums transition-colors ${flashState[toAccount] === "credit" ? "text-emerald-600" : flashState[toAccount] === "debit" ? "text-red-600" : ""}`}>
                    {balances[toAccount] !== undefined
                      ? `$${balances[toAccount].toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
                      : "..."}
                  </p>
                </div>
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                  <CreditCard className="h-5 w-5 text-primary" />
                </div>
              </div>
            </CardContent>
          </Card>
          <div className="flex justify-end -mt-2">
            <Button variant="ghost" size="sm" className="text-xs text-muted-foreground" onClick={handleResetBalances}>
              <RotateCcw className="mr-1 h-3 w-3" />
              Reset Balances
            </Button>
          </div>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          {/* Transfer Form */}
          <div className="lg:col-span-2 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Send className="h-5 w-5" />
                  New Transfer
                </CardTitle>
                <CardDescription>
                  Send money securely with durable execution guarantees
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {/* Amount Input */}
                <div className="space-y-2">
                  <Label htmlFor="amount">Amount</Label>
                  <div className="relative">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-2xl text-muted-foreground">$</span>
                    <Input
                      id="amount"
                      type="number"
                      value={amount}
                      onChange={(e) => setAmount(e.target.value)}
                      className="pl-10 text-3xl font-semibold h-16"
                      placeholder="0.00"
                      min={1}
                    />
                  </div>
                  {isLargeTransfer && (
                    <p className="text-sm text-amber-600 flex items-center gap-1">
                      <AlertTriangle className="h-4 w-4" />
                      Transfers over $1,000 require approval
                    </p>
                  )}
                </div>

                {/* From / To */}
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label>From ({SENDER_NAME})</Label>
                    <Select value={fromAccount} onValueChange={setFromAccount}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select account" />
                      </SelectTrigger>
                      <SelectContent>
                        {FROM_ACCOUNTS.map((acc) => (
                          <SelectItem key={acc.id} value={acc.name}>
                            <div className="flex items-center gap-2">
                              <Wallet className="h-4 w-4 text-muted-foreground" />
                              <span>{acc.name}</span>
                              <span className="text-muted-foreground">{acc.number}</span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {selectedFromAccount && balances[fromAccount] !== undefined && (
                      <p className="text-sm text-muted-foreground">
                        Available: ${balances[fromAccount].toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                      </p>
                    )}
                  </div>

                  <div className="space-y-2">
                    <Label>To</Label>
                    <Select value={toAccount} onValueChange={setToAccount}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select recipient" />
                      </SelectTrigger>
                      <SelectContent>
                        {TO_ACCOUNTS.map((acc) => (
                          <SelectItem key={acc.id} value={acc.name}>
                            <div className="flex items-center gap-2">
                              <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
                                {acc.initials}
                              </div>
                              <span>{acc.name}</span>
                              <span className="text-muted-foreground">{acc.number}</span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {/* Scenario Selection */}
                <div className="space-y-3">
                  <Label>Transfer Mode</Label>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                    {SCENARIOS.map((s) => (
                      <button
                        key={s.value}
                        onClick={() => setScenario(s.value)}
                        className={`rounded-lg border p-3 text-left transition-all h-[72px] flex flex-col justify-center ${
                          scenario === s.value
                            ? "border-primary bg-primary/5 ring-1 ring-primary"
                            : "border-border hover:border-primary/50 hover:bg-muted/50"
                        }`}
                      >
                        <div className="text-sm font-medium">{s.label}</div>
                        <div className="text-xs text-muted-foreground">{s.description}</div>
                      </button>
                    ))}
                  </div>
                </div>

                {/* Schedule Toggle */}
                <div className="rounded-lg border p-4">
                  <label className="flex items-start gap-3 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isSchedule}
                      onChange={(e) => setIsSchedule(e.target.checked)}
                      className="mt-1 h-4 w-4 rounded border-input text-primary focus:ring-primary"
                    />
                    <div className="space-y-1">
                      <span className="font-medium">Schedule recurring transfer</span>
                      <p className="text-sm text-muted-foreground">
                        Set up automatic transfers using Temporal Schedules
                      </p>
                    </div>
                  </label>
                  {isSchedule && (
                    <div className="mt-4 flex items-center gap-2 pl-7">
                      <span className="text-sm text-muted-foreground">Repeat every</span>
                      <Input
                        type="number"
                        value={intervalHours}
                        onChange={(e) => setIntervalHours(e.target.value)}
                        className="w-20"
                        min={1}
                      />
                      <span className="text-sm text-muted-foreground">hours</span>
                    </div>
                  )}
                </div>

                {/* Submit */}
                <Button
                  onClick={handleTransfer}
                  disabled={isLoading || !amount || Number(amount) <= 0}
                  size="lg"
                  className="w-full"
                >
                  {isLoading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Processing...
                    </>
                  ) : (
                    <>
                      Transfer ${Number(amount).toLocaleString()}
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </>
                  )}
                </Button>

                {error && (
                  <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
                    {error}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Status Sidebar */}
          <div className="space-y-6">
            {/* Active Transfer */}
            {activeWorkflowId && (
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between gap-2">
                    <CardTitle className="text-base">Transfer Status</CardTitle>
                    {statusDisplay && (
                      <Badge variant={statusDisplay.variant} className="gap-1.5">
                        <statusDisplay.icon className="h-3 w-3" />
                        {statusDisplay.label}
                      </Badge>
                    )}
                  </div>
                  <a
                    href={getWorkflowUrl(activeWorkflowId, serverInfo)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground transition-colors"
                  >
                    {activeWorkflowId}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="space-y-1">
                    <div className="flex justify-between text-xs text-muted-foreground">
                      <span>Progress</span>
                      <span>{transferStatus?.progressPercentage ?? 0}%</span>
                    </div>
                    <Progress value={transferStatus?.progressPercentage ?? 0} />
                  </div>

                  {needsApproval && (
                    <div className="rounded-md border border-amber-200 bg-amber-50 p-3">
                      <div className="flex flex-col gap-2">
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2 text-amber-700">
                            <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
                            <span className="text-xs font-medium">
                              {isLargeTransfer ? "Requires approval" : "Manual approval"}
                            </span>
                          </div>
                          <Button
                            onClick={handleApprove}
                            size="sm"
                            className="h-7 px-2.5 text-xs bg-amber-500 hover:bg-amber-600"
                          >
                            Approve
                          </Button>
                        </div>
                        {approvalStartTime && transferStatus?.approvalTime && (
                          <div className="text-amber-700">
                            <CountdownTimer
                              targetTime={new Date(approvalStartTime.getTime() + transferStatus.approvalTime * 1000)}
                              label="Expires in"
                            />
                          </div>
                        )}
                      </div>
                    </div>
                  )}

                  {/* Schedule Info */}
                  {scheduleInfo && scheduleInfo.nextRunTime && (
                    <div className="rounded-md border border-blue-200 bg-blue-50 p-3">
                      <div className="flex flex-col gap-1">
                        <span className="text-xs font-medium text-blue-700">Scheduled Transfer</span>
                        <CountdownTimer
                          targetTime={new Date(scheduleInfo.nextRunTime)}
                          label="Next run in"
                        />
                        <p className="text-xs text-blue-600 mt-1">
                          Workflow URL will be available after the schedule fires
                        </p>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Recent Transfers */}
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center justify-between">
                  Recent Transfers
                  <Button variant="ghost" size="icon" onClick={refreshWorkflows} className="h-8 w-8">
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                </CardTitle>
              </CardHeader>
              <CardContent>
                {workflows.length === 0 && schedules.length === 0 ? (
                  <div className="py-8 text-center">
                    <CreditCard className="mx-auto h-10 w-10 text-muted-foreground/50" />
                    <p className="mt-2 text-sm text-muted-foreground">No recent transfers</p>
                  </div>
                ) : (
                  <div className="space-y-2 max-h-[280px] overflow-y-auto">
                    {/* Scheduled Transfers */}
                    {schedules.map((sched) => {
                      const isActive = activeWorkflowId === sched.scheduleId;
                      return (
                        <div
                          key={sched.scheduleId}
                          className={`w-full flex items-center justify-between rounded-lg border p-3 transition-colors ${
                            isActive
                              ? "border-blue-500 bg-blue-50"
                              : "border-blue-200 bg-blue-50/50 hover:bg-blue-100/50"
                          }`}
                        >
                          <button
                            onClick={() => setActiveWorkflowId(sched.scheduleId)}
                            className="flex items-center gap-2 flex-1 text-left"
                          >
                            <Clock className="h-3.5 w-3.5 text-blue-600" />
                            <span className="font-mono text-xs truncate max-w-[100px]">
                              {sched.scheduleId.replace("schedule-", "")}
                            </span>
                          </button>
                          <div className="flex items-center gap-2">
                            <Badge variant="info" className="bg-blue-100 text-blue-700">
                              {sched.paused ? "Paused" : "Scheduled"}
                            </Badge>
                            <button
                              onClick={(e) => handleDeleteSchedule(sched.scheduleId, e)}
                              className="p-1 rounded hover:bg-red-100 text-muted-foreground hover:text-red-600 transition-colors"
                              title="Delete schedule"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          </div>
                        </div>
                      );
                    })}

                    {/* Regular Workflows */}
                    {workflows.map((wf) => {
                      // Use detailed status for active workflow, fall back to execution status
                      const isActive = activeWorkflowId === wf.workflowId;
                      const useDetailedStatus = isActive && statusDisplay;

                      const badgeVariant = useDetailedStatus
                        ? statusDisplay.variant
                        : wf.status.includes("COMPLETED")
                          ? "success"
                          : wf.status.includes("RUNNING")
                            ? "info"
                            : wf.status.includes("FAILED")
                              ? "destructive"
                              : "secondary";

                      const badgeLabel = useDetailedStatus
                        ? statusDisplay.label
                        : wf.status.replace("WORKFLOW_EXECUTION_STATUS_", "");

                      return (
                        <button
                          key={wf.workflowId}
                          onClick={() => setActiveWorkflowId(wf.workflowId)}
                          className={`w-full flex items-center justify-between rounded-lg border p-3 text-left transition-colors ${
                            isActive
                              ? "border-primary bg-primary/5"
                              : "hover:bg-muted/50"
                          }`}
                        >
                          <span className="font-mono text-xs truncate max-w-[140px]">
                            {wf.workflowId}
                          </span>
                          <Badge variant={badgeVariant}>
                            {badgeLabel}
                          </Badge>
                        </button>
                      );
                    })}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Connection Info */}
            <Card className={serverInfo?.secureConnection ? "border-emerald-200 bg-emerald-50/50" : "border-dashed"}>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  {serverInfo?.secureConnection ? (
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <CardTitle className="text-sm flex items-center gap-1.5 text-emerald-700 cursor-help">
                            <ShieldCheck className="h-4 w-4" />
                            Secure Connection
                          </CardTitle>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>mTLS</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  ) : (
                    <CardTitle className="text-sm">Connection</CardTitle>
                  )}
                  {serverInfo?.encryptPayloads && (
                    <Badge variant="success" className="text-xs">
                      Encrypted
                    </Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <dl className="grid gap-1.5 text-xs">
                  <div className="flex justify-between gap-2">
                    <dt className="text-muted-foreground">Server</dt>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <dd className="font-mono text-right cursor-help">{truncateServerAddress(serverInfo?.address)}</dd>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p className="font-mono">{serverInfo?.address}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                  <div className="flex justify-between gap-2">
                    <dt className="text-muted-foreground">Task Queue</dt>
                    <dd className="font-mono text-right">{serverInfo?.taskQueue || "..."}</dd>
                  </div>
                </dl>
              </CardContent>
            </Card>
          </div>
        </div>
      </main>
    </div>
  );
}

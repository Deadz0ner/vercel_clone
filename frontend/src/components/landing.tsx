import { CardTitle, CardDescription, CardHeader, CardContent, Card } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { useState, useRef, useCallback, useEffect } from "react"
import { GitBranch, ExternalLink, RotateCcw, Loader2, CheckCircle2, XCircle, Rocket, ArrowRight } from "lucide-react"
import axios from "axios"

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || "http://localhost:8020";
const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8020";

type DeployStatus = "idle" | "uploading" | "queued" | "building" | "uploading_artifacts" | "deployed" | "failed";

const STATUS_CONFIG: Record<DeployStatus, { label: string; color: string }> = {
  idle: { label: "Deploy", color: "" },
  uploading: { label: "Uploading", color: "text-neutral-400" },
  queued: { label: "Queued", color: "text-neutral-400" },
  building: { label: "Building", color: "text-neutral-300" },
  uploading_artifacts: { label: "Deploying", color: "text-neutral-300" },
  deployed: { label: "Deployed", color: "text-emerald-400" },
  failed: { label: "Failed", color: "text-red-400" },
};

const STEPS: DeployStatus[] = ["uploading", "queued", "building", "uploading_artifacts", "deployed"];

function StepIndicator({ status }: { status: DeployStatus }) {
  if (status === "idle" || status === "failed") return null;

  const currentIdx = STEPS.indexOf(status);

  return (
    <div className="flex items-center gap-1.5 w-full mt-6">
      {STEPS.map((step, i) => {
        const isComplete = i < currentIdx;
        const isCurrent = i === currentIdx;
        const isPending = i > currentIdx;

        return (
          <div key={step} className="flex-1 flex flex-col items-center gap-2">
            <div
              className={`h-1 w-full rounded-full transition-all duration-500 ${
                isComplete
                  ? "bg-emerald-500"
                  : isCurrent
                    ? "bg-neutral-300 animate-status-pulse"
                    : isPending
                      ? "bg-neutral-800"
                      : ""
              }`}
            />
            <span
              className={`text-[10px] uppercase tracking-wider ${
                isComplete
                  ? "text-emerald-500"
                  : isCurrent
                    ? "text-neutral-300"
                    : "text-neutral-700"
              }`}
            >
              {STATUS_CONFIG[step].label}
            </span>
          </div>
        );
      })}
    </div>
  );
}

export function Landing() {
  const [repoUrl, setRepoUrl] = useState("");
  const [uploadId, setUploadId] = useState("");
  const [status, setStatus] = useState<DeployStatus>("idle");
  const [deployedUrl, setDeployedUrl] = useState("");
  const wsRef = useRef<WebSocket | null>(null);

  const handleDeploy = useCallback(async () => {
    setStatus("uploading");
    setDeployedUrl("");
    try {
      const res = await axios.post(`${BACKEND_URL}/deploy`, {
        repoURL: repoUrl,
      });
      const projectId = res.data.id;
      setUploadId(projectId);
      setStatus("queued");

      const ws = new WebSocket(`${WS_URL}/ws/${projectId}`);
      wsRef.current = ws;

      ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        const statusStr: string = data.status;

        if (statusStr === "building") {
          setStatus("building");
        } else if (statusStr === "uploading") {
          setStatus("uploading_artifacts");
        } else if (statusStr.startsWith("deployed:")) {
          const url = statusStr.replace("deployed:", "");
          setDeployedUrl(url);
          setStatus("deployed");
          ws.close();
        } else if (statusStr === "failed") {
          setStatus("failed");
          ws.close();
        }
      };

      ws.onerror = () => {
        setStatus("failed");
      };

      ws.onclose = () => {
        wsRef.current = null;
      };
    } catch {
      setStatus("failed");
    }
  }, [repoUrl]);

  const resetState = useCallback(() => {
    setStatus("idle");
    setUploadId("");
    setDeployedUrl("");
    setRepoUrl("");
  }, []);

  useEffect(() => {
    return () => {
      wsRef.current?.close();
    };
  }, []);

  const isInProgress = status !== "idle" && status !== "deployed" && status !== "failed";
  const canDeploy = !isInProgress && repoUrl.length > 0;

  return (
    <main className="noise-bg flex flex-col items-center justify-center min-h-screen bg-background p-4">
      {/* Header */}
      <div className="mb-10 text-center relative z-10">
        <div className="flex items-center justify-center gap-3 mb-3">
          <div className="h-8 w-8 rounded-lg bg-white flex items-center justify-center">
            <Rocket className="h-4 w-4 text-black" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-white">NovaDeploy</h1>
        </div>
        <p className="text-sm text-neutral-500">Ship to production in seconds.</p>
      </div>

      {/* Main deploy card */}
      <Card className="w-full max-w-lg gradient-border bg-neutral-950/80 backdrop-blur border-neutral-800/50 relative z-10">
        <CardHeader className="pb-4">
          <CardTitle className="text-lg font-medium text-neutral-100">New Deployment</CardTitle>
          <CardDescription className="text-neutral-500">Import a GitHub repository to get started</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="github-url" className="text-neutral-400 text-xs uppercase tracking-wider">
                Repository
              </Label>
              <div className="relative">
                <GitBranch className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-600" />
                <Input
                  id="github-url"
                  value={repoUrl}
                  onChange={(e) => setRepoUrl(e.target.value)}
                  disabled={isInProgress}
                  placeholder="https://github.com/username/repo"
                  className="pl-10 bg-neutral-900 border-neutral-800 text-neutral-200 placeholder:text-neutral-600 h-11 focus-visible:ring-neutral-700 focus-visible:ring-offset-0"
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && canDeploy) handleDeploy();
                  }}
                />
              </div>
            </div>

            <Button
              onClick={handleDeploy}
              disabled={!canDeploy}
              className="w-full h-11 bg-white text-black font-medium hover:bg-neutral-200 disabled:bg-neutral-800 disabled:text-neutral-600 transition-all duration-200"
            >
              {isInProgress ? (
                <span className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {STATUS_CONFIG[status].label}...
                </span>
              ) : status === "deployed" ? (
                <span className="flex items-center gap-2">
                  Deploy Again
                  <ArrowRight className="h-4 w-4" />
                </span>
              ) : status === "failed" ? (
                <span className="flex items-center gap-2">
                  <RotateCcw className="h-4 w-4" />
                  Retry
                </span>
              ) : (
                <span className="flex items-center gap-2">
                  Deploy
                  <ArrowRight className="h-4 w-4" />
                </span>
              )}
            </Button>

            {uploadId && (status !== "idle") && (
              <p className="text-[11px] text-neutral-600 font-mono text-center">
                ID: {uploadId}
              </p>
            )}

            <StepIndicator status={status} />
          </div>
        </CardContent>
      </Card>

      {/* Success card */}
      {status === "deployed" && (
        <Card className="w-full max-w-lg mt-4 gradient-border bg-neutral-950/80 backdrop-blur border-neutral-800/50 relative z-10">
          <CardContent className="pt-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="h-8 w-8 rounded-full bg-emerald-500/10 flex items-center justify-center">
                <CheckCircle2 className="h-4 w-4 text-emerald-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-neutral-200">Deployment successful</p>
                <p className="text-xs text-neutral-500">Your site is live and ready</p>
              </div>
            </div>

            <div className="flex items-center gap-2 p-3 rounded-lg bg-neutral-900 border border-neutral-800">
              <div className="h-2 w-2 rounded-full bg-emerald-400 animate-status-pulse" />
              <code className="text-sm text-neutral-300 flex-1 truncate">{deployedUrl}</code>
              <Button
                size="sm"
                variant="ghost"
                className="h-7 px-2 text-neutral-400 hover:text-white hover:bg-neutral-800"
                onClick={() => navigator.clipboard.writeText(deployedUrl)}
              >
                Copy
              </Button>
            </div>

            <div className="flex gap-2 mt-4">
              <Button
                className="flex-1 bg-white text-black hover:bg-neutral-200"
                asChild
              >
                <a href={deployedUrl} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="h-4 w-4 mr-2" />
                  Visit Site
                </a>
              </Button>
              <Button
                className="flex-1 bg-neutral-800 text-neutral-300 hover:bg-neutral-700 border border-neutral-700"
                onClick={resetState}
              >
                <RotateCcw className="h-4 w-4 mr-2" />
                New Deploy
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Failure card */}
      {status === "failed" && (
        <Card className="w-full max-w-lg mt-4 bg-neutral-950/80 backdrop-blur border-red-900/30 relative z-10">
          <CardContent className="pt-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="h-8 w-8 rounded-full bg-red-500/10 flex items-center justify-center">
                <XCircle className="h-4 w-4 text-red-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-neutral-200">Deployment failed</p>
                <p className="text-xs text-neutral-500">Check server logs for details</p>
              </div>
            </div>
            <Button
              className="w-full bg-neutral-800 text-neutral-300 hover:bg-neutral-700 border border-neutral-700"
              onClick={resetState}
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Try Again
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Footer */}
      <p className="mt-10 text-[11px] text-neutral-700 relative z-10">
        NovaDeploy &mdash; Deploy with confidence
      </p>
    </main>
  );
}

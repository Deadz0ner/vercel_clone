import { CardTitle, CardDescription, CardHeader, CardContent, Card } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { useState, useRef, useCallback } from "react"
import axios from "axios"

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || "http://localhost:8020";
const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8020";

type DeployStatus = "idle" | "uploading" | "queued" | "building" | "uploading_artifacts" | "deployed" | "failed";

const STATUS_LABELS: Record<DeployStatus, string> = {
  idle: "Upload",
  uploading: "Uploading...",
  queued: "Queued...",
  building: "Building...",
  uploading_artifacts: "Deploying...",
  deployed: "Deployed!",
  failed: "Failed",
};

export function Landing() {
  const [repoUrl, setRepoUrl] = useState("");
  const [uploadId, setUploadId] = useState("");
  const [status, setStatus] = useState<DeployStatus>("idle");
  const [deployedUrl, setDeployedUrl] = useState("");
  const wsRef = useRef<WebSocket | null>(null);

  const handleDeploy = useCallback(async () => {
    setStatus("uploading");
    try {
      const res = await axios.post(`${BACKEND_URL}/deploy`, {
        repoURL: repoUrl,
      });
      const projectId = res.data.id;
      setUploadId(projectId);
      setStatus("queued");

      // Open WebSocket to receive real-time status updates
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

  const isInProgress = status !== "idle" && status !== "deployed" && status !== "failed";
  const canDeploy = status === "idle" || status === "failed";

  return (
    <main className="flex flex-col items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-xl">Deploy your GitHub Repository</CardTitle>
          <CardDescription>Enter the URL of your GitHub repository to deploy it</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="github-url">GitHub Repository URL</Label>
              <Input
                onChange={(e) => setRepoUrl(e.target.value)}
                placeholder="https://github.com/username/repo"
              />
            </div>
            <Button
              onClick={handleDeploy}
              disabled={!canDeploy || !repoUrl}
              className="w-full"
              type="submit"
            >
              {uploadId && isInProgress
                ? `${STATUS_LABELS[status]} (${uploadId})`
                : STATUS_LABELS[status]}
            </Button>

            {isInProgress && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <div className="h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
                {STATUS_LABELS[status]}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {status === "deployed" && (
        <Card className="w-full max-w-md mt-8">
          <CardHeader>
            <CardTitle className="text-xl">Deployment Status</CardTitle>
            <CardDescription>Your website is successfully deployed!</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <Label htmlFor="deployed-url">Deployed URL</Label>
              <Input id="deployed-url" readOnly type="url" value={deployedUrl} />
            </div>
            <br />
            <Button className="w-full" variant="outline">
              <a href={deployedUrl} target="_blank">
                Visit Website
              </a>
            </Button>
          </CardContent>
        </Card>
      )}

      {status === "failed" && (
        <Card className="w-full max-w-md mt-8 border-red-200">
          <CardContent className="pt-6">
            <p className="text-sm text-red-600">
              Deployment failed. Check the server logs for details.
            </p>
          </CardContent>
        </Card>
      )}
    </main>
  );
}

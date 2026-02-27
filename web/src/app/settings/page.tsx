"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Check } from "lucide-react";

import { api, type DataSourceInfo } from "@/lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

const PROVIDERS = [
  { id: "github", name: "GitHub" },
  { id: "wakatime", name: "Wakatime" },
];

export default function SettingsPage() {
  const router = useRouter();
  const [sources, setSources] = useState<DataSourceInfo[]>([]);
  const [loading, setLoading] = useState(true);

  const redirectToLogin = useCallback(() => {
    localStorage.removeItem("token");
    router.replace("/login");
  }, [router]);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.replace("/login");
      return;
    }
    api
      .dataSources()
      .then((data) => setSources(data.sources))
      .catch(() => redirectToLogin())
      .finally(() => setLoading(false));
  }, [router, redirectToLogin]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  const connectedProviders = new Set(sources.map((s) => s.provider));

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="flex items-center gap-4 border-b px-6 py-4">
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => router.push("/")}
        >
          <ArrowLeft />
        </Button>
        <h1 className="text-xl font-bold tracking-tight">Settings</h1>
      </header>

      <main className="flex flex-1 flex-col gap-6 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Data Sources</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-4">
              {PROVIDERS.map((provider, idx) => {
                const connected = connectedProviders.has(provider.id);
                const source = sources.find(
                  (s) => s.provider === provider.id
                );
                return (
                  <div key={provider.id} className="flex flex-col gap-4">
                    {idx > 0 && <Separator />}
                    <div className="flex items-center justify-between py-1">
                      <div className="flex items-center gap-3">
                        <span className="font-medium">{provider.name}</span>
                        {connected ? (
                          <Badge variant="default">
                            <Check
                              data-icon="inline-start"
                              className="size-3"
                            />
                            Connected
                          </Badge>
                        ) : (
                          <Badge variant="outline">Not Connected</Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {connected && source && (
                          <span className="text-xs text-muted-foreground">
                            since{" "}
                            {new Date(
                              source.connectedAt
                            ).toLocaleDateString()}
                          </span>
                        )}
                        <Button
                          variant={connected ? "destructive" : "default"}
                          size="sm"
                          disabled={!connected}
                        >
                          {connected ? "Disconnect" : "Connect"}
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}

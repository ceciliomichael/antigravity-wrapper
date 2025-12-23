"use client";

import * as React from "react";
import { useAdmin } from "../../hooks/use-admin";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { ModelSelector } from "./model-selector";
import { Plus } from "lucide-react";

export function CreateKeyForm() {
  const { createKey, isLoading, models } = useAdmin();
  const [note, setNote] = React.useState("");
  const [rateLimit, setRateLimit] = React.useState("");
  const [allowedModels, setAllowedModels] = React.useState<string[]>([]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createKey({
        note,
        rate_limit: parseInt(rateLimit) || 0,
        allowed_models: allowedModels.length > 0 ? allowedModels : undefined,
      });
      setNote("");
      setRateLimit("");
      setAllowedModels([]);
    } catch (_e) {
      // Error handled by hook
    }
  };

  return (
    <Card className="border-zinc-200 bg-white shadow-sm">
      <CardHeader className="pb-4">
        <CardTitle className="text-xl font-semibold text-zinc-900">Generate API Key</CardTitle>
        <CardDescription className="text-zinc-500">
          Create a new access key with optional rate limits and model restrictions
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Row 1: Note and Rate Limit */}
          <div className="grid gap-4 sm:grid-cols-[1fr_140px]">
            <div>
              <label className="mb-2 block text-sm font-medium text-zinc-700">
                Note
                <span className="ml-1 text-zinc-400 font-normal">(optional)</span>
              </label>
              <Input
                placeholder="e.g. Production App, Development Key"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                className="h-11 bg-white"
              />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-zinc-700">
                RPM Limit
              </label>
              <Input
                type="number"
                placeholder="0 = unlimited"
                value={rateLimit}
                onChange={(e) => setRateLimit(e.target.value)}
                className="h-11 bg-white"
              />
            </div>
          </div>

          {/* Row 2: Model Selector */}
          <div>
            <label className="mb-2 block text-sm font-medium text-zinc-700">
              Allowed Models
            </label>
            <ModelSelector
              models={models}
              selectedModels={allowedModels}
              onChange={setAllowedModels}
              showTags={true}
            />
            <p className="mt-2 text-xs text-zinc-400">
              Leave empty to allow access to all models
            </p>
          </div>

          {/* Submit Button */}
          <div className="flex justify-end pt-2">
            <Button 
              type="submit" 
              loading={isLoading} 
              className="h-11 px-6 w-full sm:w-auto"
            >
              <Plus className="mr-2 h-4 w-4" />
              Generate Key
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
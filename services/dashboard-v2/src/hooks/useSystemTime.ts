import { useState, useEffect, useCallback } from "react";
import { fetchSystemDate } from "@/services/timeService";
import { settings } from "@/config/settings";

export function useSystemTime() {
  const [systemDate, setSystemDate] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const poll = useCallback(async () => {
    try {
      const date = await fetchSystemDate();
      setSystemDate(date);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  }, []);

  useEffect(() => {
    poll();
    const id = setInterval(poll, settings.pollIntervalMs);
    return () => clearInterval(id);
  }, [poll]);

  return { systemDate, error };
}

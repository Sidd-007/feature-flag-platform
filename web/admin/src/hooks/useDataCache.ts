import { useState, useCallback, useRef } from 'react';

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  isLoading: boolean;
}

interface DataCache {
  [key: string]: CacheEntry<any>;
}

const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

export function useDataCache() {
  const [cache, setCache] = useState<DataCache>({});
  const loadingRef = useRef<Set<string>>(new Set());

  const getCacheKey = useCallback((...args: any[]) => {
    return args.filter(Boolean).join(':');
  }, []);

  const getCachedData = useCallback(<T>(key: string): T | null => {
    const entry = cache[key];
    if (!entry) return null;
    
    const isExpired = Date.now() - entry.timestamp > CACHE_DURATION;
    if (isExpired) {
      setCache(prev => {
        const newCache = { ...prev };
        delete newCache[key];
        return newCache;
      });
      return null;
    }
    
    return entry.data;
  }, [cache]);

  const setCachedData = useCallback(<T>(key: string, data: T) => {
    setCache(prev => ({
      ...prev,
      [key]: {
        data,
        timestamp: Date.now(),
        isLoading: false,
      },
    }));
    loadingRef.current.delete(key);
  }, []);

  const isDataLoading = useCallback((key: string): boolean => {
    return loadingRef.current.has(key);
  }, []);

  const setDataLoading = useCallback((key: string, loading: boolean) => {
    if (loading) {
      loadingRef.current.add(key);
      setCache(prev => ({
        ...prev,
        [key]: {
          data: prev[key]?.data || null,
          timestamp: prev[key]?.timestamp || Date.now(),
          isLoading: true,
        },
      }));
    } else {
      loadingRef.current.delete(key);
      setCache(prev => ({
        ...prev,
        [key]: {
          ...prev[key],
          isLoading: false,
        },
      }));
    }
  }, []);

  const clearCache = useCallback((pattern?: string) => {
    if (pattern) {
      setCache(prev => {
        const newCache = { ...prev };
        Object.keys(newCache).forEach(key => {
          if (key.includes(pattern)) {
            delete newCache[key];
          }
        });
        return newCache;
      });
    } else {
      setCache({});
      loadingRef.current.clear();
    }
  }, []);

  return {
    getCachedData,
    setCachedData,
    isDataLoading,
    setDataLoading,
    clearCache,
  };
}

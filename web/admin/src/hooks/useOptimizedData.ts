import { useData } from '@/providers/DataProvider';
import { useCallback, useEffect, useRef, useState } from 'react';

interface UseOptimizedDataOptions<T> {
    key: string;
    fetchFn: () => Promise<T>;
    dependencies?: any[];
    immediate?: boolean;
    onError?: (error: any) => void;
}

export function useOptimizedData<T>({
    key,
    fetchFn,
    dependencies = [],
    immediate = true,
    onError,
}: UseOptimizedDataOptions<T>) {
    const { getCachedData, setCachedData, isDataLoading, setDataLoading } = useData();
    const [data, setData] = useState<T | null>(getCachedData<T>(key));
    const [error, setError] = useState<string | null>(null);
    const fetchFnRef = useRef(fetchFn);
    const onErrorRef = useRef(onError);
    const dataFunctionsRef = useRef({ getCachedData, setCachedData, setDataLoading });

    // Update refs when props change
    fetchFnRef.current = fetchFn;
    onErrorRef.current = onError;
    dataFunctionsRef.current = { getCachedData, setCachedData, setDataLoading };

    const isLoading = isDataLoading(key);

    const fetchData = useCallback(async (force = false) => {
        // Check cache first
        const cached = dataFunctionsRef.current.getCachedData<T>(key);
        if (!force && cached) {
            setData(cached);
            return cached;
        }

        // Set loading state
        dataFunctionsRef.current.setDataLoading(key, true);
        setError(null);

        try {
            const result = await fetchFnRef.current();
            dataFunctionsRef.current.setCachedData(key, result);
            setData(result);
            return result;
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'An unexpected error occurred';
            setError(errorMessage);
            onErrorRef.current?.(err);
            throw err;
        } finally {
            dataFunctionsRef.current.setDataLoading(key, false);
        }
    }, [key]);

    // Initialize data from cache on mount
    useEffect(() => {
        const cached = getCachedData<T>(key);
        if (cached) {
            setData(cached);
        }
    }, [key, getCachedData]);

    // Fetch data when dependencies change
    useEffect(() => {
        if (immediate) {
            fetchData();
        }
    }, [immediate, fetchData, ...dependencies]);

    const refresh = useCallback(() => {
        return fetchData(true);
    }, [fetchData]);

    return {
        data,
        isLoading,
        error,
        refresh,
        fetchData,
    };
}

'use client';

import { useDataCache } from '@/hooks/useDataCache';
import React, { createContext, useContext } from 'react';

interface DataContextType {
    getCachedData: <T>(key: string) => T | null;
    setCachedData: <T>(key: string, data: T) => void;
    isDataLoading: (key: string) => boolean;
    setDataLoading: (key: string, loading: boolean) => void;
    clearCache: (pattern?: string) => void;
}

const DataContext = createContext<DataContextType | null>(null);

export function DataProvider({ children }: { children: React.ReactNode }) {
    const dataCache = useDataCache();

    return (
        <DataContext.Provider value={dataCache}>
            {children}
        </DataContext.Provider>
    );
}

export function useData() {
    const context = useContext(DataContext);
    if (!context) {
        throw new Error('useData must be used within a DataProvider');
    }
    return context;
}

'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback, useEffect } from 'react';

interface Selection {
    projectId: string | null;
    envKey: string | null;
}

export function useSelection() {
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();

    const updateSelection = useCallback(
        (updates: Record<string, string | null>) => {
            const params = new URLSearchParams(searchParams.toString());

            Object.entries(updates).forEach(([key, value]) => {
                if (value === null) {
                    params.delete(key);
                } else {
                    params.set(key, value);
                }
            });

            router.replace(`?${params.toString()}`);
        },
        [router, searchParams]
    );

    const getParam = useCallback(
        (key: string): string | null => {
            return searchParams.get(key);
        },
        [searchParams]
    );

    // Extract orgId from pathname
    const getOrgId = useCallback(() => {
        const match = pathname.match(/\/organizations\/([^\/]+)/);
        return match ? match[1] : null;
    }, [pathname]);

    // localStorage management for last selection
    const getStorageKey = useCallback((orgId: string) => {
        return `org-${orgId}-last-selection`;
    }, []);

    const getLastSelection = useCallback((orgId: string): Selection | null => {
        if (typeof window === 'undefined') return null;
        try {
            const stored = localStorage.getItem(getStorageKey(orgId));
            return stored ? JSON.parse(stored) : null;
        } catch {
            return null;
        }
    }, [getStorageKey]);

    const saveLastSelection = useCallback((orgId: string, selection: Selection) => {
        if (typeof window === 'undefined') return;
        try {
            localStorage.setItem(getStorageKey(orgId), JSON.stringify(selection));
        } catch {
            // Ignore localStorage errors
        }
    }, [getStorageKey]);

    // Get current selection from URL or localStorage
    const getCurrentSelection = useCallback((): Selection => {
        const projectId = getParam('projectId');
        const envKey = getParam('envKey');
        const gotoLast = getParam('goto') === 'last';
        const orgId = getOrgId();

        // If we have both params in URL, use them
        if (projectId && envKey) {
            return { projectId, envKey };
        }

        // If we have one param in URL, use it and clear the other
        if (projectId || envKey) {
            return { projectId, envKey };
        }

        // If goto=last is set and we have orgId, try to restore from localStorage
        if (gotoLast && orgId) {
            const lastSelection = getLastSelection(orgId);
            if (lastSelection?.projectId && lastSelection?.envKey) {
                return lastSelection;
            }
        }

        // Default: no selection
        return { projectId: null, envKey: null };
    }, [getParam, getOrgId, getLastSelection]);

    const currentSelection = getCurrentSelection();

    // Save selection to localStorage when it changes
    useEffect(() => {
        const orgId = getOrgId();
        if (orgId && currentSelection.projectId && currentSelection.envKey) {
            saveLastSelection(orgId, currentSelection);
        }
    }, [currentSelection.projectId, currentSelection.envKey, getOrgId, saveLastSelection]);

    const setSelection = useCallback(
        (selection: Partial<Selection>) => {
            const updates: Record<string, string | null> = {};
            if (selection.projectId !== undefined) {
                updates.projectId = selection.projectId;
            }
            if (selection.envKey !== undefined) {
                updates.envKey = selection.envKey;
            }
            updateSelection(updates);
        },
        [updateSelection]
    );

    const setProjectId = useCallback(
        (id: string | null) => setSelection({ projectId: id }),
        [setSelection]
    );

    const setEnvKey = useCallback(
        (key: string | null) => setSelection({ envKey: key }),
        [setSelection]
    );

    const setQuery = useCallback(
        (q: string) => updateSelection({ q: q || null }),
        [updateSelection]
    );

    const setType = useCallback(
        (type: string) => updateSelection({ type: type === 'all' ? null : type }),
        [updateSelection]
    );

    const setStatus = useCallback(
        (status: string) => updateSelection({ status: status === 'all' ? null : status }),
        [updateSelection]
    );

    const setPage = useCallback(
        (page: number) => updateSelection({ page: page === 1 ? null : page.toString() }),
        [updateSelection]
    );

    const setPageSize = useCallback(
        (size: number) => updateSelection({ pageSize: size === 25 ? null : size.toString() }),
        [updateSelection]
    );

    // Navigation helpers
    const navigateToOrgHome = useCallback(() => {
        const orgId = getOrgId();
        if (orgId) {
            router.push(`/organizations/${orgId}`);
        }
    }, [router, getOrgId]);

    const navigateToProject = useCallback((projectId: string) => {
        const orgId = getOrgId();
        if (orgId) {
            router.push(`/organizations/${orgId}/projects/${projectId}`);
        }
    }, [router, getOrgId]);

    const navigateToFlags = useCallback((projectId: string, envKey: string) => {
        const orgId = getOrgId();
        if (orgId) {
            router.push(`/organizations/${orgId}/projects/${projectId}/envs/${envKey}/flags`);
        }
    }, [router, getOrgId]);

    const navigateWithLastSelection = useCallback(() => {
        const orgId = getOrgId();
        if (orgId) {
            router.push(`/organizations/${orgId}?goto=last`);
        }
    }, [router, getOrgId]);

    return {
        // Current state
        projectId: currentSelection.projectId,
        envKey: currentSelection.envKey,
        q: getParam('q') || '',
        type: getParam('type') || 'all',
        status: getParam('status') || 'all',
        page: parseInt(getParam('page') || '1'),
        pageSize: parseInt(getParam('pageSize') || '25'),

        // Setters
        setSelection,
        setProjectId,
        setEnvKey,
        setQuery,
        setType,
        setStatus,
        setPage,
        setPageSize,
        updateSelection,

        // Navigation
        navigateToOrgHome,
        navigateToProject,
        navigateToFlags,
        navigateWithLastSelection,

        // Utilities
        getOrgId,
        getLastSelection,
        saveLastSelection,
    };
}

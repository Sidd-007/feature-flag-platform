interface ApiResponse<T = any> {
    data?: T;
    error?: string;
    status: number;
}

class ApiClient {
    private baseURL = 'http://localhost:8080/v1';
    private token: string | null = null;

    constructor() {
        // Load token from localStorage on initialization
        if (typeof window !== 'undefined') {
            this.token = localStorage.getItem('auth_token');
        }
    }

    setToken(token: string) {
        this.token = token;
        if (typeof window !== 'undefined') {
            localStorage.setItem('auth_token', token);
        }
    }

    clearToken() {
        this.token = null;
        if (typeof window !== 'undefined') {
            localStorage.removeItem('auth_token');
        }
    }

    private async request<T>(
        endpoint: string,
        options: RequestInit = {}
    ): Promise<ApiResponse<T>> {
        const url = `${this.baseURL}${endpoint}`;

        const headers: Record<string, string> = {
            'Content-Type': 'application/json',
        };

        // Add custom headers
        if (options.headers) {
            Object.assign(headers, options.headers);
        }

        if (this.token) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }

        try {
            const response = await fetch(url, {
                ...options,
                headers,
            });

            const data = response.headers.get('content-type')?.includes('application/json')
                ? await response.json()
                : await response.text();

            return {
                data: response.ok ? data : undefined,
                error: response.ok ? undefined : data.message || `HTTP ${response.status}`,
                status: response.status,
            };
        } catch (error) {
            return {
                error: error instanceof Error ? error.message : 'Network error',
                status: 0,
            };
        }
    }

    // Generic HTTP methods
    async get(url: string) {
        return this.request(url, { method: 'GET' });
    }

    async post(url: string, data?: any) {
        return this.request(url, {
            method: 'POST',
            body: data ? JSON.stringify(data) : undefined,
        });
    }

    async put(url: string, data?: any) {
        return this.request(url, {
            method: 'PUT',
            body: data ? JSON.stringify(data) : undefined,
        });
    }

    async delete(url: string) {
        return this.request(url, { method: 'DELETE' });
    }

    // Auth endpoints
    async register(data: {
        email: string;
        password: string;
        firstName: string;
        lastName: string;
    }) {
        return this.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    async login(data: { email: string; password: string }) {
        console.log('Login attempt with:', { email: data.email, password: '***' });

        const response = await this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify(data),
        });

        console.log('Login response:', { status: response.status, error: response.error, data: response.data });

        // Check for both possible token field names
        const token = (response.data as any)?.access_token || (response.data as any)?.token;
        if (token) {
            this.setToken(token);
            console.log('Token set successfully:', token.substring(0, 20) + '...');
        } else {
            console.log('No token found in response:', response.data);
        }

        return response;
    }

    async logout() {
        this.clearToken();
        return { status: 200 };
    }

    // Organizations
    async getOrganizations() {
        console.log('Getting organizations with token:', this.token ? this.token.substring(0, 20) + '...' : 'No token');
        const response = await this.request('/orgs');
        console.log('GET /orgs response:', response);
        return response;
    }

    async createOrganization(data: {
        name: string;
        description?: string;
    }) {
        // Generate slug from name with timestamp to ensure uniqueness
        let baseSlug = data.name
            .toLowerCase()
            .replace(/[^a-z0-9\s-]/g, '') // Remove special characters
            .replace(/\s+/g, '-') // Replace spaces with hyphens
            .replace(/-+/g, '-') // Replace multiple hyphens with single
            .replace(/^-|-$/g, ''); // Remove leading/trailing hyphens

        // Ensure slug is not empty
        if (!baseSlug) {
            baseSlug = 'org';
        }

        // Limit base slug length and add timestamp to make it unique
        const timestamp = Date.now().toString().slice(-6); // Last 6 digits
        const slug = `${baseSlug.substring(0, 20)}-${timestamp}`;

        const requestData = {
            name: data.name,
            slug: slug,
        };

        console.log('Creating organization with data:', requestData);

        return this.request('/orgs', {
            method: 'POST',
            body: JSON.stringify(requestData),
        });
    }

    async getOrganization(orgId: string) {
        return this.request(`/orgs/${orgId}`);
    }

    // Projects
    async getProjects(orgId: string) {
        return this.request(`/orgs/${orgId}/projects`);
    }

    async createProject(orgId: string, data: {
        name: string;
        description?: string;
        slug: string;
    }) {
        // Convert slug to key for the backend
        const requestData = {
            name: data.name,
            key: data.slug,
            description: data.description,
        };

        return this.request(`/orgs/${orgId}/projects`, {
            method: 'POST',
            body: JSON.stringify(requestData),
        });
    }

    async getProject(orgId: string, projectId: string) {
        return this.request(`/orgs/${orgId}/projects/${projectId}`);
    }

    // Environments
    async getEnvironments(orgId: string, projectId: string) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments`);
    }

    async createEnvironment(orgId: string, projectId: string, data: {
        name: string;
        key: string;
        description?: string;
    }) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    // Feature Flags
    async getFlags(orgId: string, projectId: string, envId: string) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/flags`);
    }

    async createFlag(orgId: string, projectId: string, envId: string, data: {
        key: string;
        name: string;
        description?: string;
        type: string;
        enabled: boolean;
        default_value: any;
    }) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/flags`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    async updateFlag(orgId: string, projectId: string, envId: string, flagKey: string, data: any) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/flags/${flagKey}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        });
    }

    async deleteFlag(orgId: string, projectId: string, envId: string, flagKey: string) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/flags/${flagKey}`, {
            method: 'DELETE',
        });
    }

    async publishFlag(orgId: string, projectId: string, envId: string, flagKey: string) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/flags/${flagKey}/publish`, {
            method: 'POST',
        });
    }

    async unpublishFlag(orgId: string, projectId: string, envId: string, flagKey: string) {
        return this.request(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/flags/${flagKey}/unpublish`, {
            method: 'POST',
        });
    }

    // Health check
    async getHealth() {
        return this.request('/health');
    }
}

export const apiClient = new ApiClient();
export default apiClient;

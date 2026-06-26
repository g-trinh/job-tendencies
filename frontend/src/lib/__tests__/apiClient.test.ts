import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../apiClient';

describe('apiClient — X-Active-Profile header injection', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    // Reset the module-level id before each test so tests are isolated.
    setActiveProfileId(null);
  });

  afterEach(() => {
    mock.restore();
  });

  it('omits X-Active-Profile when no id is set', async () => {
    let capturedHeaders: Record<string, string> | undefined;

    mock.onGet('/test').reply((config) => {
      capturedHeaders = config.headers as Record<string, string>;
      return [200, {}];
    });

    await apiClient.get('/test');

    expect(capturedHeaders?.['X-Active-Profile']).toBeUndefined();
  });

  it('injects X-Active-Profile when an id is set', async () => {
    setActiveProfileId('profile-abc');

    let capturedHeaders: Record<string, string> | undefined;

    mock.onGet('/test').reply((config) => {
      capturedHeaders = config.headers as Record<string, string>;
      return [200, {}];
    });

    await apiClient.get('/test');

    expect(capturedHeaders?.['X-Active-Profile']).toBe('profile-abc');
  });

  it('stops sending the header after id is cleared', async () => {
    setActiveProfileId('profile-abc');
    setActiveProfileId(null);

    let capturedHeaders: Record<string, string> | undefined;

    mock.onGet('/test').reply((config) => {
      capturedHeaders = config.headers as Record<string, string>;
      return [200, {}];
    });

    await apiClient.get('/test');

    expect(capturedHeaders?.['X-Active-Profile']).toBeUndefined();
  });
});

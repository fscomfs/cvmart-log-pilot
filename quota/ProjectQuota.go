package quota

/*
#include <stdlib.h>
#include <dirent.h>
#include <linux/fs.h>
#include <linux/quota.h>
#include <linux/dqblk_xfs.h>

#ifndef FS_XFLAG_PROJINHERIT
struct fsxattr {
__u32		fsx_xflags;
__u32		fsx_extsize;
__u32		fsx_nextents;
__u32		fsx_projid;
unsigned char	fsx_pad[12];
};
#define FS_XFLAG_PROJINHERIT	0x00000200
#endif
#ifndef FS_IOC_FSGETXATTR
#define FS_IOC_FSGETXATTR		_IOR ('X', 31, struct fsxattr)
#endif
#ifndef FS_IOC_FSSETXATTR
#define FS_IOC_FSSETXATTR		_IOW ('X', 32, struct fsxattr)
#endif

#ifndef PRJQUOTA
#define PRJQUOTA	2
#endif
#ifndef XFS_PROJ_QUOTA
#define XFS_PROJ_QUOTA	2
#endif
#ifndef Q_XSETPQLIM
#define Q_XSETPQLIM QCMD(Q_XSETQLIM, PRJQUOTA)
#endif
#ifndef Q_XGETPQUOTA
#define Q_XGETPQUOTA QCMD(Q_XGETQUOTA, PRJQUOTA)
#endif

const int Q_XGETQSTAT_PRJQUOTA = QCMD(Q_XGETQSTAT, PRJQUOTA);
*/
import "C"
import (
	"context"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type Control struct {
	backingFsBlockDev string
	prefixPath        string
	relBasePath       string
	basePath          string
	sync.RWMutex      // protect nextProjectID and quotas map
	quotas            map[string]uint32
}

type QuotaInfo struct {
	Path      string `json:"path"`
	Quota     uint64 `json:"quota"`
	UsedSpace uint64 `json:"usedSpace"`
}

type Quota struct {
	Size uint64
}
type pquotaState struct {
	sync.Mutex
	nextProjectID uint32
}

var pquotaStateInst *pquotaState
var pquotaStateOnce sync.Once

func (q *Control) SetDirQuota(targetPath string, quota Quota) error {
	log.Printf("SetDirQuota targetPath=%s relPath=%s", targetPath, q.getRealPath(targetPath))
	q.RLock()
	projectID, ok := q.quotas[q.getRealPath(targetPath)]
	q.RUnlock()
	fileInfo, err := os.Stat(q.getRealPath(targetPath))
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(q.getRealPath(targetPath), 0777)
	}
	if err == nil && !fileInfo.IsDir() {
		return fmt.Errorf("%s is not dir", targetPath)
	}

	if !ok {
		state := getPquotaState()
		state.Lock()
		projectID = state.nextProjectID

		//
		// assign project id to new container directory
		//
		err := setProjectID(q.getRealPath(targetPath), projectID)
		if err != nil {
			state.Unlock()
			return err
		}

		state.nextProjectID++
		state.Unlock()

		q.Lock()
		q.quotas[q.getRealPath(targetPath)] = projectID
		q.Unlock()
	}

	//
	// set the quota limit for the container's project id
	//
	log.Printf("SetQuota(%s, %d): projectID=%d", targetPath, quota.Size, projectID)
	return setProjectQuota(q.backingFsBlockDev, projectID, quota)
}

func (q *Control) GetDirQuota(targetPath string) (quotaInfo QuotaInfo, err error) {
	log.Printf("GetDirQuota targetPath=%s relPath=%s", targetPath, q.getRealPath(targetPath))
	q.RLock()
	projectID, ok := q.quotas[q.getRealPath(targetPath)]
	q.RUnlock()
	quotaInfo = QuotaInfo{
		Path:      targetPath,
		Quota:     0,
		UsedSpace: 0,
	}
	if !ok {
		return quotaInfo, errors.Errorf("quota not found for path: %s", targetPath)
	}
	_, err2 := os.Stat(q.getRealPath(targetPath))
	if err2 != nil && os.IsNotExist(err2) {
		return quotaInfo, errors.Errorf("quota not found for path: %s", targetPath)
	}
	//
	// get the quota limit for the container's project id
	//
	var d C.fs_disk_quota_t

	var cs = C.CString(q.backingFsBlockDev)
	defer C.free(unsafe.Pointer(cs))

	_, _, errno := unix.Syscall6(unix.SYS_QUOTACTL, C.Q_XGETPQUOTA,
		uintptr(unsafe.Pointer(cs)), uintptr(C.__u32(projectID)),
		uintptr(unsafe.Pointer(&d)), 0, 0)
	if errno != 0 {
		return quotaInfo, errors.Wrapf(errno, "Failed to get quota limit for projid %d on %s",
			projectID, q.backingFsBlockDev)
	}
	quotaInfo.Quota = uint64(d.d_blk_hardlimit) * 512
	quotaInfo.UsedSpace = uint64(d.d_bcount) * 512

	return quotaInfo, nil
}

func (q *Control) ReleaseDir(targetPath string) error {
	log.Printf("ReleaseDir targetPath=%s relPath=%s", targetPath, q.getRealPath(targetPath))
	q.RLock()
	_, ok := q.quotas[q.getRealPath(targetPath)]
	q.RUnlock()
	if !ok {
		return errors.Errorf("quota not found for path: %s", targetPath)
	}
	if filepath.Dir(targetPath) != q.basePath {
		return errors.Errorf("can not release path: %s", targetPath)
	}
	return os.RemoveAll(q.getRealPath(targetPath))
}

func (q *Control) GetNodeSpaceInfo(targetPath string) (quotaInfo QuotaInfo, err error) {
	log.Printf("GetNodeSpaceInfo targetPath=%s relPath=%s", targetPath, q.getRealPath(targetPath))
	var stat syscall.Statfs_t
	quotaInfo = QuotaInfo{Path: targetPath}
	err = syscall.Statfs(q.getRealPath(targetPath), &stat)
	if err != nil {
		return quotaInfo, fmt.Errorf("Error getting filesystem stats:%s", err.Error())
	}

	available := stat.Bavail * uint64(stat.Bsize)
	total := stat.Blocks * uint64(stat.Bsize)
	quotaInfo.Quota = total
	quotaInfo.UsedSpace = total - available
	return quotaInfo, nil
}

func (q *Control) GetImageDiskQuotaInfo(imageName string, dockerClient *docker.Client) (quotaInfo QuotaInfo, err error) {
	log.Printf("GetImageDiskQuotaInfo imageName=%s", imageName)
	var stat syscall.Statfs_t
	quotaInfo = QuotaInfo{Path: ""}
	var imageID string
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err == nil {
		for _, image := range images {
			for _, tag := range image.RepoTags {
				if tag == imageName {
					imageID = image.ID
					break
				}
			}
			if imageID != "" {
				break
			}
		}
	}

	if imageID == "" {
		return quotaInfo, fmt.Errorf("image not fount")
	}
	var targetPath string
	imgInspcet, _, err := dockerClient.ImageInspectWithRaw(context.Background(), imageID)
	if err == nil {
		for key, val := range imgInspcet.GraphDriver.Data {
			if key == "UpperDir" {
				targetPath = val
			}
		}
	}
	if targetPath == "" {
		return quotaInfo, fmt.Errorf("targetPath not fount")
	}
	relPath, e := FindRealPath(q.prefixPath, targetPath)
	if e != nil {
		return quotaInfo, e
	}
	quotaInfo.Path = relPath
	err = syscall.Statfs(quotaInfo.Path, &stat)
	if err != nil {
		return quotaInfo, fmt.Errorf("Error getting filesystem stats:%s", err.Error())
	}
	available := stat.Bavail * uint64(stat.Bsize)
	total := stat.Blocks * uint64(stat.Bsize)
	quotaInfo.Quota = total
	quotaInfo.UsedSpace = total - available
	return quotaInfo, nil
}

func openDir(path string) (*C.DIR, error) {
	Cpath := C.CString(path)
	defer free(Cpath)

	dir := C.opendir(Cpath)
	if dir == nil {
		return nil, errors.Errorf("failed to open dir: %s", path)
	}
	return dir, nil
}
func closeDir(dir *C.DIR) {
	if dir != nil {
		C.closedir(dir)
	}
}
func getDirFd(dir *C.DIR) uintptr {
	return uintptr(C.dirfd(dir))
}
func setProjectID(targetPath string, projectID uint32) error {
	dir, err := openDir(targetPath)
	if err != nil {
		return err
	}
	defer closeDir(dir)

	var fsx C.struct_fsxattr
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, getDirFd(dir), C.FS_IOC_FSGETXATTR,
		uintptr(unsafe.Pointer(&fsx)))
	if errno != 0 {
		return errors.Wrapf(errno, "failed to get projid for %s", targetPath)
	}
	fsx.fsx_projid = C.__u32(projectID)
	fsx.fsx_xflags |= C.FS_XFLAG_PROJINHERIT
	_, _, errno = unix.Syscall(unix.SYS_IOCTL, getDirFd(dir), C.FS_IOC_FSSETXATTR,
		uintptr(unsafe.Pointer(&fsx)))
	if errno != 0 {
		return errors.Wrapf(errno, "failed to set projid for %s", targetPath)
	}

	return nil
}
func free(p *C.char) {
	C.free(unsafe.Pointer(p))
}
func hasQuotaSupport(backingFsBlockDev string) (bool, error) {
	var cs = C.CString(backingFsBlockDev)
	defer free(cs)
	var qstat C.fs_quota_stat_t

	_, _, errno := unix.Syscall6(unix.SYS_QUOTACTL, uintptr(C.Q_XGETQSTAT_PRJQUOTA), uintptr(unsafe.Pointer(cs)), 0, uintptr(unsafe.Pointer(&qstat)), 0, 0)
	if errno == 0 && qstat.qs_flags&C.FS_QUOTA_PDQ_ENFD > 0 && qstat.qs_flags&C.FS_QUOTA_PDQ_ACCT > 0 {
		return true, nil
	}

	switch errno {
	// These are the known fatal errors, consider all other errors (ENOTTY, etc.. not supporting quota)
	case unix.EFAULT, unix.ENOENT, unix.ENOTBLK, unix.EPERM:
	default:
		return false, nil
	}

	return false, errno
}

// getProjectID - get the project id of path on xfs
func getProjectID(targetPath string) (uint32, error) {
	dir, err := openDir(targetPath)
	if err != nil {
		return 0, err
	}
	defer closeDir(dir)

	var fsx C.struct_fsxattr
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, getDirFd(dir), C.FS_IOC_FSGETXATTR,
		uintptr(unsafe.Pointer(&fsx)))
	if errno != 0 {
		return 0, errors.Wrapf(errno, "failed to get projid for %s", targetPath)
	}

	return uint32(fsx.fsx_projid), nil
}

// makeBackingFsDev gets the backing block device of the driver home directory
// and creates a block device node under the home directory to be used by
// quotactl commands.
func makeBackingFsDev(home string) (string, error) {
	var stat unix.Stat_t
	if err := unix.Stat(home, &stat); err != nil {
		return "", err
	}

	backingFsBlockDev := path.Join(home, "backingFsBlockDev2")
	// Re-create just in case someone copied the home directory over to a new device
	unix.Unlink(backingFsBlockDev)
	err := unix.Mknod(backingFsBlockDev, unix.S_IFBLK|0600, int(stat.Dev))
	switch err {
	case nil:
		return backingFsBlockDev, nil

	case unix.ENOSYS, unix.EPERM:
		return "", fmt.Errorf("Not suppored")

	default:
		return "", errors.Wrapf(err, "failed to mknod %s", backingFsBlockDev)
	}
}

// findNextProjectID - find the next project id to be used for containers
// by scanning driver home directory to find used project ids
func (q *Control) findNextProjectID(home string, baseID uint32) error {
	state := getPquotaState()
	state.Lock()
	defer state.Unlock()

	checkProjID := func(path string) (uint32, error) {
		projid, err := getProjectID(path)
		if err != nil {
			return projid, err
		}
		if projid > 0 {
			q.quotas[path] = projid
		}
		if state.nextProjectID <= projid {
			state.nextProjectID = projid + 1
		}
		return projid, nil
	}

	files, err := ioutil.ReadDir(home)
	if err != nil {
		return errors.Errorf("read directory failed: %s", home)
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		path := filepath.Join(home, file.Name())
		projid, err := checkProjID(path)
		if err != nil {
			return err
		}
		if projid > 0 && projid != baseID {
			continue
		}
		subfiles, err := ioutil.ReadDir(path)
		if err != nil {
			return errors.Errorf("read directory failed: %s", path)
		}
		for _, subfile := range subfiles {
			if !subfile.IsDir() {
				continue
			}
			subpath := filepath.Join(path, subfile.Name())
			_, err := checkProjID(subpath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// setProjectQuota - set the quota for project id on xfs block device
func setProjectQuota(backingFsBlockDev string, projectID uint32, quota Quota) error {
	var d C.fs_disk_quota_t
	d.d_version = C.FS_DQUOT_VERSION
	d.d_id = C.__u32(projectID)
	d.d_flags = C.XFS_PROJ_QUOTA

	d.d_fieldmask = C.FS_DQ_BHARD | C.FS_DQ_BSOFT
	d.d_blk_hardlimit = C.__u64(quota.Size / 512)
	d.d_blk_softlimit = d.d_blk_hardlimit

	var cs = C.CString(backingFsBlockDev)
	defer C.free(unsafe.Pointer(cs))

	_, _, errno := unix.Syscall6(unix.SYS_QUOTACTL, C.Q_XSETPQLIM,
		uintptr(unsafe.Pointer(cs)), uintptr(d.d_id),
		uintptr(unsafe.Pointer(&d)), 0, 0)
	if errno != 0 {
		return errors.Wrapf(errno, "failed to set quota limit for projid %d on %s",
			projectID, backingFsBlockDev)
	}

	return nil
}

func getPquotaState() *pquotaState {
	pquotaStateOnce.Do(func() {
		pquotaStateInst = &pquotaState{
			nextProjectID: 1,
		}
	})
	return pquotaStateInst
}

func NewControl(prefixPath string, basePath string) (*Control, error) {
	relBasePath, err := FindRealPath(prefixPath, basePath)
	if err != nil {
		if os.IsNotExist(err) {
			e := os.MkdirAll(path.Join("/", prefixPath, filepath.Dir(basePath)), 0777)
			if e == nil {
				part, e := GetXFSMount(prefixPath)
				if e == nil && part.Mountpoint != "" {
					log.Printf("xfs disk mount pointer %+v", part)
					disk_path := path.Join(part.Mountpoint, "cloud-disk")
					if _, e := os.Stat(disk_path); e != nil {
						if e := os.MkdirAll(disk_path, 0777); e != nil {
							log.Errorf("make dir err %s", err.Error())
						}
					}
					if prefixPath == "" {
						log.Printf("create symlink %s to %s", disk_path, basePath)
						if e := os.Symlink(disk_path, basePath); e != nil {
							return nil, e
						}
					} else {
						log.Printf("create symlink %s to %s", strings.TrimPrefix(disk_path, path.Join("/", prefixPath)), path.Join("/", prefixPath, basePath))
						if e := os.Symlink(strings.TrimPrefix(disk_path, path.Join("/", prefixPath)), path.Join("/", prefixPath, basePath)); e != nil {
							return nil, e
						}
					}
					relBasePath, err = FindRealPath(prefixPath, basePath)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, e
				}
			} else {
				return nil, e
			}
		} else {
			return nil, err
		}
	}

	log.Printf("real path = %s", relBasePath)
	//
	// create backing filesystem device node
	//
	backingFsBlockDev, err := makeBackingFsDev(relBasePath)
	if err != nil {
		return nil, err
	}

	// check if we can call quotactl with project quotas
	// as a mechanism to determine (early) if we have support
	hasQuotaSupport, err := hasQuotaSupport(backingFsBlockDev)
	if err != nil {
		return nil, err
	}
	if !hasQuotaSupport {
		return nil, fmt.Errorf("Not supported")
	}

	//
	// Get project id of parent dir as minimal id to be used by driver
	//
	baseProjectID, err := getProjectID(relBasePath)
	if err != nil {
		return nil, err
	}
	minProjectID := baseProjectID + 10000

	//
	// Test if filesystem supports project quotas by trying to set
	// a quota on the first available project id
	//
	quota := Quota{
		Size: 0,
	}
	if err := setProjectQuota(backingFsBlockDev, minProjectID, quota); err != nil {
		return nil, err
	}

	q := Control{
		backingFsBlockDev: backingFsBlockDev,
		quotas:            make(map[string]uint32),
		basePath:          basePath,
		relBasePath:       relBasePath,
		prefixPath:        prefixPath,
	}

	//
	// update minimum project ID
	//
	state := getPquotaState()
	state.updateMinProjID(minProjectID)

	//
	// get first project id to be used for next container
	//
	err = q.findNextProjectID(relBasePath, baseProjectID)
	if err != nil {
		return nil, err
	}

	log.Printf("NewControl(%s): realPath = %s  nextProjectID = %d", basePath, relBasePath, state.nextProjectID)
	return &q, nil
}

func (c *Control) getRealPath(targetPath string) string {
	return strings.Replace(path.Join("/", targetPath), c.basePath, c.relBasePath, 1)
}

func (state *pquotaState) updateMinProjID(minProjectID uint32) {
	state.Lock()
	defer state.Unlock()
	if state.nextProjectID <= minProjectID {
		state.nextProjectID = minProjectID + 1
	}
}

func FindRealPath(prefixPath string, spath string) (relPath string, err error) {
	if prefixPath != "" && !strings.HasPrefix(prefixPath, "/") {
		prefixPath = "/" + prefixPath
	}
	paths := strings.FieldsFunc(spath, func(c rune) bool {
		return c == '/'
	})
	currentDir := path.Join("/", prefixPath)
	for i := 0; i < len(paths); i++ {
		currentDir = path.Join(currentDir, paths[i])
		if fileInfo, err2 := os.Lstat(currentDir); err2 == nil {
			if fileInfo.Mode()&os.ModeSymlink != 0 {
				if relPath, err2 := os.Readlink(currentDir); err2 != nil {
					return "", err2
				} else {
					var rel string
					if !strings.HasPrefix(relPath, "/") { //
						rel = path.Join(filepath.Dir(currentDir), relPath)
					} else {
						rel = path.Join("/", prefixPath, relPath)
					}
					if _, err2 := os.Stat(rel); err2 == nil {
						currentDir = rel
					} else {
						return rel, err2
					}
				}
			} else {
				if i == len(paths) {
					return currentDir, nil
				}
			}
		} else {
			return "", err2
		}
	}
	return currentDir, nil
}

func GetXFSMount(prefixPath string) (maxPart MaxSpacePartition, err error) {
	parts, e := disk.Partitions(true)
	if e != nil {
		return maxPart, e
	}
	xfsParts := []MaxSpacePartition{}
	for _, part := range parts {
		if part.Fstype == "xfs" {
			if prefixPath != "" {
				if !strings.HasPrefix(part.Mountpoint, path.Join("/", prefixPath)) {
					continue
				}
				if fi, e := os.Stat(part.Mountpoint); e == nil {
					if !fi.IsDir() {
						continue
					}
				}
			}
			usag, e := disk.Usage(part.Mountpoint)
			if e != nil {
				continue
			}
			p := MaxSpacePartition{
				Device:     part.Device,
				Opts:       part.Opts,
				Fstype:     part.Fstype,
				Mountpoint: part.Mountpoint,
				Total:      usag.Total,
				DirLength:  len(strings.Split(part.Mountpoint, "/")),
			}
			xfsParts = append(xfsParts, p)
		}
	}
	maxPart = MaxSpacePartition{
		DirLength: 100,
	}
	log.Printf("all device %+v", xfsParts)
	for i, _ := range xfsParts {
		if xfsParts[i].Total >= maxPart.Total && xfsParts[i].Total > 0 {
			if xfsParts[i].Device == maxPart.Device {
				if maxPart.DirLength > xfsParts[i].DirLength {
					maxPart = xfsParts[i]
				}
			} else {
				maxPart = xfsParts[i]
			}

		}
	}
	return maxPart, nil
}

type MaxSpacePartition struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype"`
	Opts       string `json:"opts"`
	Total      uint64 `json:"total"`
	DirLength  int    `json:"dir_length"`
}

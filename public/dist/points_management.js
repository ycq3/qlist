document.addEventListener('DOMContentLoaded', function() {
    const pointsForm = document.getElementById('pointsForm');
    const userPointsForm = document.getElementById('userPointsForm');
    const usersTableBody = document.querySelector('#usersTable tbody');
    const pointsTableBody = document.querySelector('#pointsTable tbody');
    const logContainer = document.getElementById('pointsLog');

    // 处理积分配置表单提交
    pointsForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const formData = {
            fileUrl: pointsForm.fileUrl.value,
            points: parseInt(pointsForm.points.value),
            description: pointsForm.description.value
        };

        fetch('/api/configurePoints', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('保存失败');
            }
            return response.json();
        })
        .then(data => {
            showNotification('配置保存成功', 'success');
            pointsForm.reset();
            loadPointsList();
        })
        .catch(error => {
            showNotification(error.message, 'error');
        });
    });

    // 处理用户积分操作表单提交
    userPointsForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const formData = {
            username: userPointsForm.username.value,
            points: parseInt(userPointsForm.points.value),
            description: userPointsForm.description.value
        };

        fetch('/api/adminGrantPoints', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('积分操作失败');
            }
            return response.json();
        })
        .then(data => {
            showNotification('积分操作成功', 'success');
            userPointsForm.reset();
            loadUsersList();
            loadPointsLog();
        })
        .catch(error => {
            showNotification(error.message, 'error');
        });
    });

    // 加载用户列表
    function loadUsersList() {
        fetch('/api/getUsersList')
            .then(response => response.json())
            .then(data => {
                usersTableBody.innerHTML = '';
                data.users.forEach(user => {
                    const row = document.createElement('tr');
                    row.className = 'hover:bg-gray-50';
                    // 格式化 createdAt 日期
                    const createdAt = new Date(user.createdAt).toLocaleString();
                    row.innerHTML = `
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${user.username}</td>
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${user.provider}</td>
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${createdAt}</td>
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${user.points}</td>
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            <button onclick="fillUserForm('${user.username}')" 
                                    class="text-blue-600 hover:text-blue-900 mr-3">积分操作</button>
                        </td>
                    `;
                    usersTableBody.appendChild(row);
                });
            })
            .catch(error => {
                usersTableBody.innerHTML = `
                    <tr>
                        <td colspan="5" class="px-6 py-4 text-center text-sm text-red-500">
                            加载失败：${error.message}
                        </td>
                    </tr>
                `;
            });
    }

    // 加载积分配置列表
    function loadPointsList() {
        fetch('/api/getPointsList')
            .then(response => response.json())
            .then(data => {
                pointsTableBody.innerHTML = '';
                data.data.forEach(item => {
                    const row = document.createElement('tr');
                    row.className = 'hover:bg-gray-50';
                    row.innerHTML = `
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${item.fileUrl}</td>
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${item.points}</td>
                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${item.description || '-'}</td>
                    `;
                    pointsTableBody.appendChild(row);
                });
            })
            .catch(error => {
                pointsTableBody.innerHTML = `
                    <tr>
                        <td colspan="5" class="px-6 py-4 text-center text-sm text-red-500">
                            加载失败：${error.message}
                        </td>
                    </tr>
                `;
            });
    }

    // 加载积分日志
    function loadPointsLog() {
        fetch('/api/getPointsLog')
            .then(response => response.json())
            .then(data => {
                logContainer.innerHTML = '';
                data.logs.forEach(log => {
                    const logItem = document.createElement('div');
                    logItem.className = 'p-3 bg-gray-50 rounded-lg';
                    logItem.innerHTML = `
                        <div class="flex justify-between items-center">
                            <div>
                                <span class="text-sm font-medium">${log.userId}</span>
                                <span class="text-sm text-gray-500 mx-2">-</span>
                                <span class="text-sm">${log.description || log.fileUrl}</span>
                            </div>
                            <span class="text-sm font-medium ${log.points >= 0 ? 'text-green-600' : 'text-red-600'}">${
                                log.points >= 0 ? '+' : ''}${log.points}分</span>
                        </div>
                        <div class="text-xs text-gray-500 mt-1">${new Date(log.createdAt).toLocaleString()}</div>
                    `;
                    logContainer.appendChild(logItem);
                });
            })
            .catch(error => {
                logContainer.innerHTML = `
                    <div class="p-3 text-center text-sm text-red-500">
                        加载失败：${error.message}
                    </div>
                `;
            });
    }

    // 显示通知
    function showNotification(message, type) {
        const alert = document.createElement('div');
        alert.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg ${
            type === 'success' ? 'bg-green-500' : 'bg-red-500'} text-white`;
        alert.textContent = message;
        document.body.appendChild(alert);
        setTimeout(() => alert.remove(), 3000);
    }

    // 填充用户表单
    window.fillUserForm = function(username) {
        userPointsForm.username.value = username;
        userPointsForm.points.focus();
    };

    // 初始加载
    loadUsersList();
    loadPointsList();
    loadPointsLog();
});
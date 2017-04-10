/* Digital Rebar: Provision */
/* Copyright 2016, RackN */
/* License: Apache v2 */
/* jshint esversion: 6 */

class Subnet extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.handleOptionChange = this.handleOptionChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var subnet = this.props.subnet;
    subnet._expand = !subnet._expand;
    this.props.change(this.props.index, subnet);
  }

  // gets the name of an option from its code
  getCodeName(code) {
    var codes = {
      1: "Subnet Mask",
      3: "Default Gateway",
      6: "DNS Server",
      15: "Domain Name",
      28: "Broadcast",
      42: "NTP Server",
      67: "Bootfile Name"
    };
    return codes[code] || "Code " + code;
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    if(event.target.type === "number" && val && typeof val !== 'undefined')
      val = parseInt(val);
    if(event.target.type === "select-one") {
      val = val === "true";
    }
    var subnet = this.props.subnet;
    subnet[event.target.name] = val;
    subnet._edited = true;

    this.props.change(this.props.index, subnet);
  }

  // called when an option input is changed
  handleOptionChange(event) {
    var subnet = this.props.subnet;
    subnet.Options[event.target.name].Value = event.target.value;
    subnet._edited = true;

    this.props.change(this.props.index, subnet);
  }

  // renders the element
  render() {
    var subnet = JSON.parse(JSON.stringify(this.props.subnet));
    return (
      <tbody 
        className={
          (subnet.updating ? 'updating-content' : '') + " " + (subnet._expand ? "expanded" : "")}
        style={{
          position: "relative",
          backgroundColor: (subnet._error ? '#fdd' : (subnet._new ? "#dfd" : (subnet._edited ? "#eee" : "#fff"))),
          borderBottom: "thin solid #ddd"
        }}>
        <tr>
          <td>
            {subnet._new ? <input
              type="text"
              name="Name"
              size="8"
              placeholder="eth0"
              value={subnet.Name}
              onChange={this.handleChange}/> : subnet.Name}
          </td>
          <td>
            <input
              type="text"
              name="Subnet"
              size="15"
              placeholder="192.168.1.1/24"
              value={subnet.Subnet}
              onChange={this.handleChange}/>
          </td>
          <td>
            <select
              name="OnlyReservations"
              type="bool"
              value={subnet.OnlyReservations}
              onChange={this.handleChange}>
                <option value="true">Required</option>
                <option value="false">Optional</option>
              </select>
          </td>
          <td>
            <input
              type="number"
              name="ActiveLeaseTime"
              style={{width: "50px"}}
              placeholder="0"
              value={subnet.ActiveLeaseTime}
              onChange={this.handleChange}/>
            &nbsp;&amp;&nbsp;
            <input
              type="number"
              name="ReservedLeaseTime"
              style={{width: "50px"}}
              placeholder="7200"
              value={subnet.ReservedLeaseTime}
              onChange={this.handleChange}/>
              &nbsp;seconds
          </td>
          <td>
            <input
              type="text"
              name="ActiveStart"
              size="12"
              placeholder="10.0.0.0"
              value={subnet.ActiveStart}
              onChange={this.handleChange}/>
            ...
            <input
              type="text"
              name="ActiveEnd"
              size="12"
              placeholder="10.0.0.255"
              value={subnet.ActiveEnd}
              onChange={this.handleChange}/>
          </td>
          <td>
            {subnet._new ? <button onClick={this.update}>Add</button> :
            (subnet._edited ? <button onClick={this.update}>Update</button> : '')}
            <button onClick={this.remove}>Remove</button>
            <button onClick={this.props.copy}>Copy</button>
          </td>
        </tr>
        <tr>
          <td colSpan="7">
            {subnet._expand ? (<div>
              <h2>Other Values</h2>
              <table>
                <tbody>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>Next Server</td>
                    <td>
                      <input
                        type="text"
                        name="NextServer"
                        size="12"
                        value={subnet.NextServer}
                        onChange={this.handleChange}/>
                    </td>
                  </tr>
                </tbody>
              </table>
              <h2>Options</h2>
              <table>
                <tbody>
                  {subnet.Options.map((val, i) =>
                    <tr key={i}>
                      <td style={{textAlign: "right", fontWeight: "bold"}}>{this.getCodeName(val.Code)}</td>
                      <td>
                        <input
                          type="text"
                          name={i}
                          value={val.Value}
                          onChange={this.handleOptionChange}/>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>): <span/>}
            {subnet._error && <div>
              <h2>{subnet._errorMessage}</h2>
            </div>}
            <div className="expand" onClick={this.toggleExpand}>
              {subnet._expand ? <span>&#x25B4;</span> : <span>&#x25BE;</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class Subnets extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      subnets: [],
      interfaces: [],
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.addSubnet = this.addSubnet.bind(this);
    this.updateSubnet = this.updateSubnet.bind(this);
    this.removeSubnet = this.removeSubnet.bind(this);
    this.changeSubnet = this.changeSubnet.bind(this);
  }
  
  // gets the subnet and interface json from the api
  getSubnets() {
    return new Promise((resolve, reject) => {
      var subnets = {};
      var interfaces = {};

      // get the interfaces from the api
      $.getJSON("../api/v3/interfaces", data => {
        for(var key in data) {
          var iface = data[key];
          interfaces[iface.Name] = iface;
        }

        // add get subnets from the api and update the state
        $.getJSON("../api/v3/subnets", data => {
          for(var key in data) {
            var subnet = data[key];
            subnets[subnet.Name] = subnet;
            // don't show interfaces if the name matches
            if (interfaces[subnet.Name] != undefined)
              delete interfaces[subnet.Name];
          }

          resolve({
            interfaces: interfaces,
            subnets: subnets
          });

        }).fail(() => {
          reject("Failed getting subnets");
        });

      }).fail(() => {
        reject("Failed getting interfaces");
      });
    });
  }

  // get the subnets and interfaces once this component mounts
  componentDidMount() {
    this.getSubnets().then(data => {
      this.setState({
        subnets: Object.keys(data.subnets).map(k => data.subnets[k]),
        interfaces: Object.keys(data.interfaces).map(k => data.interfaces[k])
      }, err => {
        // rejected ?? 
      });
    });
  }

  // called to create a new subnet
  // allows some data other than defaults to be passed in
  addSubnet(template) {
    var subnet = {
      _new: true,
      Name: '',
      ActiveLeaseTime: 60,
      ReservedLeaseTime: 7200,
      OnlyReservations: false,
      ActiveStart: '',
      Subnet: '',
      ActiveEnd: '',
      Strategy: "MAC",
      Options: [
        {Code: 3, Value: ''},
        {Code: 6, Value: ''},
        {Code: 15, Value: ''},
        {Code: 67, Value: ''}
      ]
    };

    // merge the template into our subnet if we have one
    if(typeof template !== "undefined") {
      for(var key in template) {
        if(key[0] === "_")
          continue;

        if(key === 'Options') {
          for(var i = 0; i < template.Options.length; i++) {
            var index = [3, 6, 15, 67].indexOf(template.Options[i].Code);
            if(index >= 0) {
              subnet.Options[index].Value = template.Options[i].Value;
            }
            else {
              subnet.Options.push(template.Options[i]);
            }
          }
        } else
          subnet[key] = template[key];
      }
    }

    // update the state
    this.setState({
      subnets: this.state.subnets.concat(subnet)
    });
  }

  // makes the post/put request to update the subnet
  // also updates the interface
  updateSubnet(i) {
    var subnet = this.state.subnets[i];
    subnet.updating = true;
    this.setState({subnet: this.state.subnets});

    $.ajax({
      type: (subnet._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets" + (subnet._new ? "" : "/" + subnet.Name),
      data: JSON.stringify(subnet)
    }).done((resp) => {
      
      // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);

      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      subnets[i] = resp;
      this.setState({
        subnets: subnets
      });

    }).fail((err) => {
      
      var subnets = this.state.subnets.concat([]);
      var subnet = subnets[i];
      subnet.updating = false;
      subnet._error = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        subnet._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        subnet._errorMessage = err.status;
      }

      this.setState({
        subnets: subnets
      });
    });
  }

  // makes the delete request to remove the subnet or just deletes the new subnet
  removeSubnet(i) {
    var subnets = this.state.subnets.concat([]);
    var subnet = this.state.subnets[i];
    if(subnet._new) {
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });
      return;
    }
    subnets[i].updating = true;
    this.setState({subnets: subnets});

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets/" + subnet.Name,
    }).done((resp) => {
            // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });

    }).fail((err) => {
      subnet.updating = false;
      subnet._error = true;
      // If our error is from the backend
      if(err.responseText) {      
        var response = JSON.parse(err.responseText);
        subnet._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        subnet._errorMessage = err.status;
      }

      this.setState({
        subnets: subnets
      });
    });
  }

  // updates the state and changes a subnet at a specified index
  changeSubnet(i, subnet) {
    var subnets = this.state.subnets.concat([]);
    subnets[i] = subnet;
    this.setState({
      subnets: subnets
    });
  }

  render() {
    $('#subnetCount').text(this.state.subnets.length);
    return (
    <div>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span>Subnets</span>
        <span>
          <a target="_blank" href="http://rocket-skates.readthedocs.io/en/latest/doc/ui.html#subnets">UI Help</a> | <a target="_blank" href="/swagger-ui/#/subnets">API Help</a>
        </span>
      </h2>

      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Name/NIC</th>
            <th>Subnet</th>
            <th>Reservations</th>
            <th>Active &amp; Reserved Lease</th>
            <th>Range</th>
            <th></th>
          </tr>
        </thead>
        {this.state.subnets.map((val, i) =>
          <Subnet
            subnet={val}
            update={this.updateSubnet}
            change={this.changeSubnet}
            remove={this.removeSubnet}
            copy={()=>this.addSubnet(this.state.subnets[i])}
            key={i}
            index={i} />
        )}
        <tfoot>
          <tr>
            <td colSpan="7" style={{textAlign: "center"}}>
              <button onClick={()=>this.addSubnet({})}>New Subnet</button>
            </td>
          </tr>
        </tfoot>
      </table>

      <h2 style={{display: "flex", flexDirection: "row", alignItems: "center"}}>
        <span>Available Interfaces: </span>
        <span className="interface-list">
          {this.state.interfaces.map((val) => 
            val.Addresses.map((subval, i) =>
              <a key={val.Name+"-"+i} className="interface-pair" onClick={()=>this.addSubnet({Name: val.Name, Subnet: subval})}>
                <header>{val.Name}</header>
                <subhead>{subval}</subhead>
              </a>
            )
          )}
        </span>
      </h2>
    </div>
    );
  }
}


class Token extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      token: "",
      code: 1
    };

    this.handleChange = this.handleChange.bind(this);
    this.setToken = this.setToken.bind(this);
  }
  
  // gets the name of an option from its code
  getCodeName() {
    var codes = {
      0: "API Unreachable",
      1: " ",
      200: "Granted",
      401: "No Token",
      403: "Invalid Token"
    };
    return codes[this.state.code] || "Code " + this.state.code;
  }

  // get the page and interfaces once this component mounts
  componentDidMount() {
    if (location.search.startsWith("?token=")) {
      var t = location.search.substring(7);
      this.setState({token: t});
      this.setToken(t);
    }
  }

  setToken(token) {
    $.ajaxSetup({
      headers : {
        'Authorization' : 'Bearer ' + token
      }
    });
    // get the interfaces from the api
    $.getJSON("../api/v3/prefs", data => { 
      this.setState({code: 200, token: token});
      this.props.onAccessChange(true);
    }).fail((jqXHR, textStatus, errorThrown) => {
      this.setState({code: jqXHR.status, token: token});
      this.props.onAccessChange(false);
    });
  }

  // called when an input changes
  handleChange(event) {
    this.setToken(event.target.value);
  }

  render() {
    return (
      <div>
        Access Token: 
        <input
          type="text"
          name="token"
          size="15"
          placeholder="no_token"
          value={this.state.token}
          onChange={this.handleChange} />
        <strong>{this.getCodeName()}</strong>
      </div>
    );
  }
}

class Machine extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var machine = this.props.machine;
    machine._expand = !machine._expand;
    this.props.change(this.props.index, machine);
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    var machine = this.props.machine;
    machine[event.target.name] = val;
    machine._edited = true;

    this.props.change(this.props.index, machine);
  }

  // renders the element
  render() {
    var machine = JSON.parse(JSON.stringify(this.props.machine));
    return (
      <tbody 
        className={(machine.updating ? 'updating-content' : '') + " " + (machine._expand ? "expanded" : "")}
        style={{
          position: "relative",
          backgroundColor: (machine._error ? '#fdd' : (machine._new ? "#dfd" : (machine._edited ? "#eee" : "#fff"))),
          borderBottom: "thin solid #ddd"
        }}>
        <tr>
          <td>
            <input
              type="text"
              name="Name"
              size="15"
              placeholder="name.example.com"
              value={machine.Name}
              onChange={this.handleChange} />
          </td>
          <td>
            <input
              type="text"
              name="Address"
              size="15"
              placeholder="0.0.0.0"
              value={machine.Address}
              onChange={this.handleChange} />
          </td>
          <td>
            <select
              name="BootEnv"
              type="bool"
              value={machine.BootEnv}
              onChange={this.handleChange}>
                { this.props.bootenvs.map((val) =>
                  <option key={val} value={val}>{val}</option>
                )}
            </select>
          </td>
          <td>
            <input
              type="text"
              name="Description"
              size="15"
              placeholder=""
              value={machine.Description}
              onChange={this.handleChange} />            
          </td>
          <td>
            {( machine.Uuid ?
              <div title={machine.Uuid}>
                {machine.Uuid.slice(0,4)}...{machine.Uuid.slice(-4)}
              </div>
              : "not set" )}
          </td>
          <td>
            {machine._new ? <button onClick={this.update}>Add</button> :
            (machine._edited ? <button onClick={this.update}>Update</button> : '')}
            <button onClick={this.remove}>Remove</button>
          </td>
        </tr>
        <tr>
          <td colSpan="6">
            {machine._expand ? (<div>
              {machine._error && <div>
                <h2>API Error: {machine._errorMessage}</h2>
              </div>}
              <h2>Template Errors</h2>
              {(machine.Errors ? machines.Errors : "none.")}
              <h2>Template Params</h2>
              {(machine.Params ? machines.Params : "none.")}
            </div>): <span/>}
            <div className="expand" onClick={this.toggleExpand}>
              {machine._expand ? <span>&#x25B4;</span> : <span>&#x25BE;</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class Machines extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      machines: [],
      // hack for now.  ideally, we'd pull this from the bootenvs!
      bootenvs: []
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.addMachine = this.addMachine.bind(this);
    this.changeMachine = this.changeMachine.bind(this);
    this.removeMachine = this.removeMachine.bind(this);
    this.updateMachine = this.updateMachine.bind(this);
  }
  
  // gets the machine json from the api
  getMachines() {
    return new Promise((resolve, reject) => {
      var bootenvs = [];

      // get the interfaces from the api
      $.getJSON("../api/v3/machines", data => {

        // add get bootenvs from the api and update the state
        // header Authorization:Bearer [string]
        $.getJSON("../api/v3/bootenvs", data2 => {
          for(var key in data2) {
            if (data2[key].Available)
              bootenvs.push(data2[key].Name)
          }
          resolve({
            machines: data,
            bootenvs: bootenvs,
          });
        });

      }).fail(() => {
        reject("Failed getting Machines");
      });

    });
  }

  // get the machine once this component mounts
  componentDidMount() {
    this.getMachines().then(data => {
      this.setState({
        machines: data.machines,
        bootenvs: data.bootenvs
      }, err => {
        // rejected ?? 
      });
    });
  }

  // called to create a new machine
  // allows some data other than defaults to be passed in
  addMachine() {
    var machine = {
      Name: "",
      Address: "0.0.0.0",
      BootEnv: "ignore",
      Description: "",
      Uuid: null,
      _new: true
    };
    // update the state
    this.setState({
      machines: this.state.machines.concat(machine)
    });
  }

  // makes the post/put request to update the machine
  updateMachine(i) {
    var machine = this.state.machines[i];
    machine.updating = true;
    this.setState({machine: this.state.machines});

    $.ajax({
      type: (machine._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/machines" + (machine._new ? "" : "/" + machine.Uuid),
      data: JSON.stringify(machine)
    }).done((resp) => {
      
      // update the machines list with our new machine
      var machines = this.state.machines.concat([]);
      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      machines[i] = resp;
      this.setState({
        machines: machines
      });

    }).fail((err) => {
      
      var machines = this.state.machines.concat([]);
      var machine = machines[i];
      machine.updating = false;
      machine._error = true;
      machine._expand = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        machine._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        machine._errorMessage = err.status;
      }

      this.setState({
        machines: machines
      });
    });
  }

  removeMachine(i) {
    var machines = this.state.machines.concat([]);
    var machine = this.state.machines[i];
    if (machine._new) {
      machines.splice(i,1);
      this.setState({machines: machines});
      return;
    }
    machines[i].updating = true;

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/machines/" + machine.Uuid,
    }).done((resp) => {
            // update the subnets list with our new interface
      var machines = this.state.machines.concat([]);
      machines.splice(i, 1);
      this.setState({
        machines: machines
      });

    }).fail((err) => {
      machine.updating = false;
      machine._error = true;
      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        machine._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        machine._errorMessage = err.status;
      }

      this.setState({
        machines: machines
      });
    });
  }

  // updates the state and changes a machine at a specified index
  changeMachine(i, machine) {
    var machines = this.state.machines.concat([]);
    machines[i] = machine;
    this.setState({
      machines: machines
    });
  }

  render() {
    $('#machineCount').text(this.state.machines.length);
    return (
    <div>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span>Machines</span>
        <span>
          <a target="_blank" href="http://rocket-skates.readthedocs.io/en/latest/doc/ui.html#machines">UI Help</a> | <a target="_blank" href="/swagger-ui/#/machines">API Help</a>
        </span>
      </h2>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Address</th>
            <th>BootEnv</th>
            <th>Description</th>
            <th>Uuid</th>
            <th></th>
          </tr>
        </thead>
        {this.state.machines.map((val, i) =>
          <Machine
            machine={val}
            bootenvs={this.state.bootenvs}
            update={this.updateMachine}
            change={this.changeMachine}
            remove={this.removeMachine}
            key={i}
            index={i}
          />
        )}
        <tfoot>
          <tr>
            <td colSpan="6" style={{textAlign: "center"}}>
              <button onClick={()=>this.addMachine({})}>New Machine</button>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    );
  }
}

class BootEnv extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
    this.changeTemplate = this.changeTemplate.bind(this);
    this.addTemplate = this.addTemplate.bind(this);
    this.removeTemplate = this.removeTemplate.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var bootenv = this.props.bootenv;
    bootenv._expand = !bootenv._expand;
    this.props.change(this.props.index, bootenv);
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    if(event.target.type === "number" && val && typeof val !== 'undefined')
      val = parseInt(val);
    var bootenv = this.props.bootenv;
    bootenv[event.target.name] = val;
    bootenv._edited = true;

    this.props.change(this.props.index, bootenv);
  }

  changeTemplate(event, i) {
    this.props.bootenv.Templates[i][event.target.name] = event.target.value;
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }
  
  addTemplate(event) {
    this.props.bootenv.Templates.push({
      Name: "",
      Path: "",
      ID: ""
    });
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }

  removeTemplate(event, i) {
    this.props.bootenv.Templates.splice(i, 1);
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }


  // renders the element
  render() {
    var bootenv = JSON.parse(JSON.stringify(this.props.bootenv));
    return (
      <tbody 
        className={(bootenv.updating ? 'updating-content' : '') + " " + (bootenv._expand ? "expanded" : "")}
        style={{
          position: "relative",
          backgroundColor: (bootenv._error ? '#fdd' : (bootenv._new ? "#dfd" : (bootenv._edited ? "#eee" : "#fff"))),
          borderBottom: "thin solid #ddd"
        }}>
        <tr>
          <td>
            {bootenv._new ? <input
              type="text"
              name="Name"
              size="8"
              placeholder="eth0"
              value={bootenv.Name}
              onChange={this.handleChange}/> : bootenv.Name}
          </td>
          <td>
            {bootenv.Available ? "Yes" : "Error"}
          </td>
          <td>
            <a href={bootenv.OS.IsoUrl}>{bootenv.OS.IsoFile}</a>
          </td>
          <td>
            {bootenv._new ? <button onClick={this.update}>Add</button> :
            (bootenv._edited ? <button onClick={this.update}>Update</button> : '')}
            <button onClick={this.remove}>Remove</button>
            <button onClick={this.props.copy}>Copy</button>
          </td>
        </tr>
        <tr>
          <td colSpan="7">
            {bootenv._expand ? (<div>
              <table>
                <tbody>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>
                      Iso Name
                    </td>
                    <td>
                      <input
                        type="text"
                        size="30"
                        value={bootenv.OS.IsoFile}
                        onChange={(event) => {
                          bootenv._edited = true;
                          bootenv.OS.IsoFile = event.target.value;
                          this.props.change(this.props.index, bootenv);
                        }}/>
                    </td>
                  </tr>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>
                      Iso URL
                    </td>
                    <td>
                      <input
                        type="text"
                        size="30"
                        value={bootenv.OS.IsoUrl}
                        onChange={(event) => {
                          bootenv._edited = true;
                          bootenv.OS.IsoUrl = event.target.value;
                          this.props.change(this.props.index, bootenv);
                        }}/>
                    </td>
                  </tr>

                </tbody>
              </table>
              
              <h2>Templates</h2>
              <table>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Path</th>
                    <th>ID</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {bootenv.Templates.map((val, i) =>
                    <tr key={i}>
                      <td>
                        <input
                          type="text"
                          size="15"
                          name="Name"
                          value={val.Name}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <input
                          type="text"
                          size="30"
                          name="Path"
                          value={val.Path}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <input
                          type="text"
                          size="20"
                          name="ID"
                          value={val.ID}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <button onClick={(e)=>this.removeTemplate(e, i)}>Remove</button>
                      </td>
                    </tr>
                  )}
                  <tr>
                    <td colSpan="4" style={{textAlign: "center"}}>
                      <button onClick={this.addTemplate}>Add Template</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>): <span/>}
            {bootenv._error && <div>
              <h2>{bootenv._errorMessage}</h2>
            </div>}
            <div className="expand" onClick={this.toggleExpand}>
              {bootenv._expand ? <span>&#x25B4;</span> : <span>&#x25BE;</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class BootEnvs extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      bootenvs: []
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.addBootEnv = this.addBootEnv.bind(this);
    this.updateBootEnv = this.updateBootEnv.bind(this);
    this.removeBootEnv = this.removeBootEnv.bind(this);
    this.changeBootEnv = this.changeBootEnv.bind(this);
  }
  
  // gets the bootenv json from the api
  getBootEnvs() {
    return new Promise((resolve, reject) => {

      // get the interfaces from the api
      $.getJSON("../api/v3/bootenvs", data => {
        resolve({
          bootenvs: data,
        });
      }).fail(() => {
        reject("Failed getting BootEnvs");
      });

    });
  }

  // get the bootenvs once this component mounts
  componentDidMount() {
    this.getBootEnvs().then(data => {
      this.setState({
        bootenvs: data.bootenvs,
      }, err => {
        // rejected ?? 
      });
    });
  }

  // called to create a new subnet
  // allows some data other than defaults to be passed in
  addBootEnv(template) {
    var bootenv = {
      _new: true,
      Name: '',
      Description: '',
      OS: {
        Name: '',
        Family: '',
        Codename: '',
        Version: '',
        IsoFile: '',
        IsoSha256: '',
        IsoUrl: ''
      },
      Templates: [],
      Kernel: "",
      Initrds: [],
      RequiredParams: [],
      Available: true,
      Errors: []
    };

    // merge the template into our bootenv if we have one
    if(typeof template !== "undefined") {
      for(var key in template) {
        if(key[0] === "_")
          continue;

        bootenv[key] = template[key];
      }
    }

    // update the state
    this.setState({
      bootenvs: this.state.bootenvs.concat(bootenv)
    });
  }

  // makes the post/put request to update the bootenv
  updateBootEnv(i) {
    var bootenv = this.state.bootenvs[i];
    bootenv.updating = true;
    this.setState({bootenv: this.state.bootenvs});

    $.ajax({
      type: (bootenv._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/bootenvs" + (bootenv._new ? "" : "/" + bootenv.Name),
      data: JSON.stringify(bootenv)
    }).done((resp) => {
      
      // update the bootenvs list with our new interface
      var bootenvs = this.state.bootenvs.concat([]);

      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      bootenvs[i] = resp;
      this.setState({
        bootenvs: bootenvs
      });

    }).fail((err) => {
      
      var bootenvs = this.state.bootenvs.concat([]);
      var bootenv = bootenvs[i];
      bootenv.updating = false;
      bootenv._error = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        bootenv._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        bootenv._errorMessage = err.status;
      }

      this.setState({
        bootenvs: bootenvs
      });
    });
  }

  // makes the delete request to remove the subnet or just deletes the new subnet
  removeBootEnv(i) {
    var subnets = this.state.subnets.concat([]);
    var subnet = this.state.subnets[i];
    if(subnet._new) {
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });
      return;
    }
    subnets[i].updating = true;
    this.setState({subnets: subnets});

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets/" + subnet.Name,
    }).done((resp) => {
            // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });

    }).fail((err) => {
      subnet.updating = false;
      subnet._error = true;
      // If our error is from the backend
      if(err.responseText) {      
        var response = JSON.parse(err.responseText);
        subnet._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        subnet._errorMessage = err.status;
      }

      this.setState({
        subnets: subnets
      });
    });
  }

  // updates the state and changes a subnet at a specified index
  changeBootEnv(i, bootenv) {
    var bootenvs = this.state.bootenvs.concat([]);
    bootenvs[i] = bootenv;
    this.setState({
      bootenvs: bootenvs
    });
  }

  render() {
    $('#bootenvCount').text(this.state.bootenvs.length);
    return (
    <div>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span>Boot Environments</span>
        <span>
          <a target="_blank" href="http://rocket-skates.readthedocs.io/en/latest/doc/ui.html#bootenvs">UI Help</a> | <a target="_blank" href="/swagger-ui/#/bootenvs">API Help</a>
        </span>
      </h2>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Environment</th>
            <th>Available</th>
            <th>Path</th>
            <th></th>
          </tr>
        </thead>
        {this.state.bootenvs.map((val, i) =>
          <BootEnv
            bootenv={val}
            update={this.updateBootEnv}
            change={this.changeBootEnv}
            remove={this.removeBootEnv}
            copy={()=>this.addBootEnv(this.state.bootenvs[i])}
            key={i}
            index={i} />
        )}
        <tfoot>
          <tr>
            <td colSpan="4" style={{textAlign: "center"}}>
              <button onClick={()=>this.addBootEnv({})}>New BootEnv</button>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    );
  }
}

class Page extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      access: false
    };

    this.onAccessChange = this.onAccessChange.bind(this);
    //this.handleChange = this.handleChange.bind(this);
  }
  
  // get the page and interfaces once this component mounts
  componentDidMount() {
  }

  // called when an input changes
  onAccessChange(access) {
    this.setState({access: access});
  }

  render() {
    const access = this.state.access;
    if (access) {
      return (
        <div id="swagger-ui-container" className="swagger-ui-wrap">
          <Subnets
            style={{paddingTop: '51px'}}
            onAccessChange={this.onAccessChange} />
          <hr/>
          <BootEnvs
            style={{paddingTop: '51px'}}
            onAccessChange={this.onAccessChange} />
          <hr/>
          <Machines
            style={{paddingTop: '51px'}}
            onAccessChange={this.onAccessChange} />
        </div>
      );
    }
    return (
      <div>
        <center><Token access={access} onAccessChange={this.onAccessChange} /></center>
      </div>
    );
  }
}

ReactDOM.render(<Page />, page);
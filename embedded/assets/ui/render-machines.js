/* Digital Rebar: Provision */
/* Copyright 2016, RackN */
/* License: Apache v2 */
/* jshint esversion: 6 */

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
          <a target="_blank" href="http://provision.readthedocs.io/en/latest/doc/ui.html#machines">UI Help</a> | <a target="_blank" href="/swagger-ui/#/machines">API Help</a>
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

ReactDOM.render(<Machines />, machines);

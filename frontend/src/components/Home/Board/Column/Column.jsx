/* eslint-disable react/jsx-props-no-spreading */
import React from 'react';
import PropTypes from 'prop-types';
import { Droppable } from 'react-beautiful-dnd';
import { Col } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faPlusCircle } from '@fortawesome/free-solid-svg-icons';

import Task from './Task/Task';
import { columnOrder } from './columnOrder';
import { capFirstLetterOf } from '../../../../misc/utils';

import './column.sass';
import window from '../../../../misc/window';

const Column = ({
  id, name, tasks, handleActivate,
}) => (
  <Col className="Col" xs={3}>
    <div className={`Column ${capFirstLetterOf(name)}Column`}>
      <div className="Header">{name.toUpperCase()}</div>

      <Droppable droppableId={id.toString()}>
        {(provided) => (
          <div
            className="Body"
            {...provided.droppableProps}
            ref={provided.innerRef}
          >
            {tasks
              .sort((task) => task.order)
              .map((task) => (
                <Task
                  id={task.id}
                  title={task.title}
                  description={task.description}
                  order={task.order}
                  handleActivate={handleActivate}
                />
              ))}

            {provided.placeholder}

            {name === columnOrder.INBOX && (
              <button
                className="CreateButton"
                onClick={handleActivate(window.CREATE_TASK)}
                type="button"
              >
                <FontAwesomeIcon className="Icon" icon={faPlusCircle} />
              </button>
            )}
          </div>
        )}
      </Droppable>
    </div>
  </Col>
);

Column.propTypes = {
  id: PropTypes.string.isRequired,
  name: PropTypes.string.isRequired,
  tasks: PropTypes.arrayOf({
    id: PropTypes.number.isRequired,
    title: PropTypes.string.isRequired,
    description: PropTypes.string.isRequired,
    order: PropTypes.number.isRequired,
  }).isRequired,
  handleActivate: PropTypes.func,
};

Column.defaultProps = {
  handleActivate: () => console.log('Cannot create task here.'),
};

export default Column;